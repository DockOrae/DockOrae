#!/bin/bash
###############################################################################
#
# Docker Manager Go 一键安装/管理脚本
#
# Version: 2.0.0
# Last Updated: 2026-08-28
#
# Description:
#   面向 Docker Manager (https://github.com/MinimaxFlora/Docker_Manager_Go)
#   的一键脚本,提供:
#     - 两种安装方式:Docker Compose 安装 / 本地二进制安装(systemd)
#     - 未安装 Docker 时自动安装(Debian/Ubuntu amd64/arm64 最新稳定版)
#     - 域名绑定:acme.sh 自动申请 Let's Encrypt 证书(参考 3x-ui)
#     - 自动检测国内/海外网络分流,国内使用加速源/镜像拉取
#     - 安装、更新、卸载、启停、备份恢复、密码重置等管理功能
#
# Requirements:
#   - Linux + root
#   - Docker Compose 方式需要 Docker(未安装会自动装)
#   - 二进制方式无需 Docker
#
# Usage:
#   ./install.sh                      交互菜单
#   ./install.sh install              安装(--force 覆盖重装;DM_MODE=compose|binary 指定方式)
#   ./install.sh ssl                  证书管理(域名绑定/acme.sh)
#   ./install.sh update               更新
#   ./install.sh uninstall            卸载
#   ./install.sh start|stop|restart|status
#   ./install.sh backup|restore       备份/恢复
#   ./install.sh reset-passwd         重置密码
#   ./install.sh info                 安装信息
#
# Env:
#   DM_PORT        面板端口(默认 8080)
#   DM_DATA_DIR    数据目录(默认 /opt/docker-manager/data)
#   DM_INSTALL_DIR 安装目录(默认 /opt/docker-manager)
#   DM_IMAGE       镜像(默认 zhaoweiwen123/docker-manager-go:latest)
#   DM_MODE        安装方式 compose|binary
#   DM_PRIVILEGED  特权模式 true/false(仅 compose,默认 false)
#
###############################################################################

RED_COLOR='\e[1;31m'
GREEN_COLOR='\e[1;32m'
YELLOW_COLOR='\e[1;33m'
BLUE_COLOR='\e[1;34m'
CYAN_COLOR='\e[1;36m'
RES='\e[0m'

# ---------- 默认配置 ----------
DM_PORT="${DM_PORT:-8080}"
DM_INSTALL_DIR="${DM_INSTALL_DIR:-/opt/docker-manager}"
DM_DATA_DIR="${DM_DATA_DIR:-$DM_INSTALL_DIR/data}"
DM_CERT_DIR="${DM_CERT_DIR:-$DM_INSTALL_DIR/cert}"
DM_IMAGE="${DM_IMAGE:-zhaoweiwen123/docker-manager-go:latest}"
DM_PRIVILEGED="${DM_PRIVILEGED:-false}"
DM_MODE="${DM_MODE:-}"
CONTAINER_NAME="docker-manager"
BIN_NAME="docker-manager-go"
COMPOSE_FILE="$DM_INSTALL_DIR/docker-compose.yml"
SERVICE_FILE="$DM_INSTALL_DIR/docker-manager.service"
BACKUP_DIR="$DM_INSTALL_DIR/backups"
GITHUB_REPO="MinimaxFlora/Docker_Manager_Go"

# 国内镜像加速源(compose 拉镜像用,按顺序尝试)
CN_MIRRORS=(
  "docker.1panel.live"
  "dockerpull.org"
  "hub.rat.dev"
)
# 国内 GitHub 加速(二进制下载用,按顺序尝试)
CN_GH_PROXIES=(
  "https://ghproxy.net/"
  "https://gh-proxy.com/"
  "https://ghfast.top/"
)

log_info()  { echo -e "${GREEN_COLOR}[INFO]${RES} $*"; }
log_warn()  { echo -e "${YELLOW_COLOR}[WARN]${RES} $*"; }
log_error() { echo -e "${RED_COLOR}[ERROR]${RES} $*"; }
log_step()  { echo -e "${BLUE_COLOR}==>${RES} $*"; }

die() { log_error "$*"; exit 1; }

confirm() {
  local msg="$1"
  local choice
  read -r -p "$(echo -e "${YELLOW_COLOR}$msg [y/N]${RES} ")" choice
  [[ "$choice" == "y" || "$choice" == "Y" ]]
}

is_domain() {
  [[ "$1" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$ ]]
}

# ---------- 域名解析校验(仿 3x-ui ssl_cert_issue:申请证书前必须先确认 A 记录指向本机) ----------
# 校验 IPv4 是否为可路由的公网地址(排除内网/保留段,含 28.0.0.0/8 DoD 保留段等)
is_public_ip() {
  local ip="$1" a b c d
  [[ "$ip" =~ ^[0-9]+\.([0-9]+\.){2}[0-9]+$ ]] || return 1
  IFS='.' read -r a b c d <<< "$ip"
  [ "$a" -le 255 ] && [ "$b" -le 255 ] && [ "$c" -le 255 ] && [ "$d" -le 255 ] || return 1
  [ "$a" -eq 0 ] && return 1                 # 0.0.0.0/8
  [ "$a" -eq 10 ] && return 1                # 内网 10/8
  [ "$a" -eq 28 ] && return 1                # DoD 保留段(本次事故 IP 来源)
  [ "$a" -eq 100 ] && [ "$b" -ge 64 ] && [ "$b" -le 127 ] && return 1  # CGNAT 100.64/10
  [ "$a" -eq 127 ] && return 1               # loopback
  [ "$a" -eq 169 ] && [ "$b" -eq 254 ] && return 1  # link-local
  [ "$a" -eq 172 ] && [ "$b" -ge 16 ] && [ "$b" -le 31 ] && return 1   # 内网 172.16/12
  [ "$a" -eq 192 ] && [ "$b" -eq 168 ] && return 1                     # 内网 192.168/16
  [ "$a" -eq 198 ] && { [ "$b" -eq 18 ] || [ "$b" -eq 19 ]; } && return 1
  [ "$a" -eq 203 ] && [ "$b" -eq 0 ] && [ "$c" -eq 113 ] && return 1
  return 0
}

# 获取本机公网 IP:优先 DM_PUBLIC_IP 手动指定(VPS 控制台公网 IP 为准),
# 否则多源查询并校验必须是公网地址;全部失败返回非 0。
get_public_ip() {
  if [ -n "$DM_PUBLIC_IP" ]; then
    if is_public_ip "$DM_PUBLIC_IP"; then
      echo "$DM_PUBLIC_IP"
      return 0
    fi
    log_warn "DM_PUBLIC_IP=$DM_PUBLIC_IP 不是有效公网 IP,忽略"
  fi
  local ip=""
  for src in "https://api.ipify.org" "https://ifconfig.me/ip" "https://ip.sb" "https://ipinfo.io/ip"; do
    ip=$(timeout 5 curl -s4 --connect-timeout 4 "$src" 2>/dev/null | grep -oE '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | head -1)
    if [ -n "$ip" ] && is_public_ip "$ip"; then
      echo "$ip"
      return 0
    fi
  done
  return 1
}

# 解析域名 A 记录(多源:getent → nslookup → dig)
resolve_domain_ip() {
  local domain="$1" ip=""
  ip=$(getent ahostsv4 "$domain" 2>/dev/null | awk '{print $1}' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | head -1)
  [ -z "$ip" ] && ip=$(timeout 5 nslookup -type=A "$domain" 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -vE '^(0|127|255)\.' | head -1)
  [ -z "$ip" ] && ip=$(timeout 5 dig +short "$domain" A 2>/dev/null | grep -E '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$' | head -1)
  echo "$ip"
}

# 校验域名 A 记录 == 本机公网 IP,不一致直接终止(3x-ui check_domain 同款逻辑)
check_domain_dns() {
  local domain="$1"
  log_step "校验域名解析(仿 3x-ui:必须指向本机公网 IP)..."
  local pub_ip dns_ip
  pub_ip=$(get_public_ip) || {
    log_error "无法获取本机公网 IP(外部查询服务不可达或返回非公网地址)"
    log_error "请在 VPS 控制台查看公网 IP,然后手动指定重试: DM_PUBLIC_IP=<公网IP> bash install.sh ssl"
    return 1
  }
  dns_ip=$(resolve_domain_ip "$domain")
  if [ -z "$dns_ip" ]; then
    log_error "无法解析域名 $domain 的 A 记录"
    log_error "请到 DNS 服务商(如 Cloudflare)添加 A 记录: $domain → $pub_ip"
    return 1
  fi
  if ! is_public_ip "$dns_ip"; then
    log_error "域名 ${domain} 解析到非公网地址 ${dns_ip}(内网/保留段),请检查 DNS 记录值是否正确"
    log_error "正确值应为本机公网 IP: ${pub_ip}"
    return 1
  fi
  if [ "$dns_ip" != "$pub_ip" ]; then
    log_error "域名 ${domain} 当前解析到 ${dns_ip},而本机公网 IP 是 ${pub_ip}!"
    log_error "请到 DNS 服务商把 A 记录改为 ${pub_ip},等解析全球生效(dnschecker.org 可确认)后重新执行本命令"
    return 1
  fi
  log_info "域名解析校验通过: ${domain} → ${dns_ip} ✓"
}

# ---------- 系统检测 ----------
check_env() {
  [ "$(uname -s)" != "Linux" ] && die "此脚本仅支持 Linux 系统"
  [ "$(id -u)" -ne 0 ] && die "请使用 root 权限运行: sudo bash install.sh"
  command -v curl >/dev/null 2>&1 || die "未检测到 curl,请先安装: apt install curl -y"
  command -v tar >/dev/null 2>&1 || die "未检测到 tar"
}

# ---------- 网络检测(国内/海外分流) ----------
detect_network() {
  local CN=0
  if timeout 3 curl -sI https://www.google.com >/dev/null 2>&1; then
    CN=0
  elif timeout 3 curl -sI https://www.baidu.com >/dev/null 2>&1; then
    CN=1
  else
    if timeout 5 curl -sI https://registry-1.docker.io/v2/ >/dev/null 2>&1; then
      CN=0
    else
      CN=1
    fi
  fi
  if [ "$CN" -eq 1 ]; then
    log_info "检测到网络环境: ${YELLOW_COLOR}国内${RES},将使用加速源"
    USE_MIRROR=1
  else
    log_info "检测到网络环境: ${GREEN_COLOR}海外/直连${RES}"
    USE_MIRROR=0
  fi
}

# ---------- Docker 检测 / 自动安装 ----------
install_docker_if_missing() {
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    log_info "Docker 已安装且运行正常"
    docker compose version >/dev/null 2>&1 || log_warn "Docker Compose 插件未安装,Compose 方式将不可用"
    return 0
  fi
  if command -v docker >/dev/null 2>&1; then
    log_warn "Docker 已安装但服务未运行,尝试启动..."
    systemctl start docker 2>/dev/null || service docker start 2>/dev/null || true
    sleep 2
    docker info >/dev/null 2>&1 && { log_info "Docker 已启动"; return 0; }
    die "Docker 服务无法启动,请手动排查: journalctl -u docker"
  fi
  log_warn "未检测到 Docker,即将自动安装(Debian/Ubuntu amd64/arm64 默认最新稳定版)..."
  confirm "是否继续自动安装 Docker?" || die "请手动安装 Docker 后重试"
  if [ "$USE_MIRROR" -eq 1 ]; then
    log_step "国内源安装 Docker..."
    sh -c "$(curl -fsSL https://testingcf.jsdelivr.net/gh/MinimaxFlora/Docker_Private_Source@master/install.sh)" || \
      sh -c "$(curl -fsSL https://github.com/MinimaxFlora/Docker_Private_Source/raw/refs/heads/master/install.sh)" || \
      die "Docker 自动安装失败,请手动安装后重试"
  else
    log_step "官方源安装 Docker..."
    sh -c "$(curl -fsSL https://github.com/MinimaxFlora/Docker_Private_Source/raw/refs/heads/master/install.sh)" || \
      sh -c "$(curl -fsSL https://testingcf.jsdelivr.net/gh/MinimaxFlora/Docker_Private_Source@master/install.sh)" || \
      die "Docker 自动安装失败,请手动安装后重试"
  fi
  command -v docker >/dev/null 2>&1 || die "Docker 安装失败,请手动安装后重试"
  # 确保服务启动 + socket 就绪(刚安装的 Docker daemon 可能未完全起来)
  systemctl enable docker >/dev/null 2>&1 || true
  systemctl start docker >/dev/null 2>&1 || service docker start >/dev/null 2>&1 || true
  log_step "等待 Docker 服务就绪..."
  for i in $(seq 1 15); do
    [ -S /var/run/docker.sock ] && docker info >/dev/null 2>&1 && break
    sleep 1
  done
  docker info >/dev/null 2>&1 || die "Docker 服务未就绪,请手动排查: systemctl status docker"
  log_info "Docker 就绪: $(docker --version)"
}

# ---------- 架构检测(二进制安装) ----------
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l|armhf) echo "armv7" ;;
    armv6l) echo "armv6" ;;
    armv5l) echo "armv5" ;;
    i386|i686) echo "386" ;;
    s390x) echo "s390x" ;;
    *) die "不支持的架构: $(uname -m)" ;;
  esac
}

# ---------- 镜像拉取(国内走加速源) ----------
pull_image() {
  local image="$DM_IMAGE"
  if [ "$USE_MIRROR" -eq 1 ]; then
    local pulled=0
    for mirror in "${CN_MIRRORS[@]}"; do
      log_step "尝试通过加速源 $mirror 拉取..."
      if docker pull "$mirror/$image" >/dev/null 2>&1; then
        docker tag "$mirror/$image" "$image" >/dev/null 2>&1
        docker rmi "$mirror/$image" >/dev/null 2>&1
        log_info "加速源拉取成功: $mirror"
        pulled=1
        break
      fi
    done
    [ "$pulled" -eq 0 ] && {
      log_warn "加速源均失败,尝试直连拉取..."
      docker pull "$image" || die "镜像拉取失败: docker pull $image"
    }
  else
    docker pull "$image" || die "镜像拉取失败: docker pull $image"
  fi
}

# ---------- compose 文件生成 ----------
generate_compose() {
  mkdir -p "$DM_INSTALL_DIR" "$DM_DATA_DIR" "$DM_CERT_DIR"
  log_step "生成 docker-compose.yml: $COMPOSE_FILE"
  cat > "$COMPOSE_FILE" <<EOF
services:
  ${CONTAINER_NAME}:
    image: ${DM_IMAGE}
    container_name: ${CONTAINER_NAME}
    restart: unless-stopped
    ports:
      - "${DM_PORT}:8080"
    environment:
      - DATA_DIR=/data
      - PORT=8080
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ${DM_DATA_DIR}:/data
      - ${DM_CERT_DIR}:/data/cert:ro
      - /:/host:ro
      - /etc/docker:/host/etc/docker:ro
    privileged: ${DM_PRIVILEGED}
EOF
  log_info "compose 文件已生成(端口 ${DM_PORT},数据目录 ${DM_DATA_DIR},证书目录 ${DM_CERT_DIR})"
}

# ---------- 设置面板 HTTPS(监听域名 + 证书路径) ----------
# 注意:面板设置实际存储在 SQLite(settings.json 只是首次启动的迁移源,
# 直接改 settings.json 无效!)—— 必须通过面板 API:
# 登录 → GET 现有设置 → sed 替换三个字段 → PUT 全量 → 重启
set_panel_https() {
  local domain="$1" cert="$2" key="$3"
  local mode
  mode=$(read_mode)
  if [ "$mode" = "compose" ]; then
    if set_panel_https_via_container "$domain" "$cert" "$key"; then
      log_info "面板已配置: 监听域名 ${domain},证书 ${cert} / ${key}(通过面板 API 写入)"
      return 0
    fi
  else
    if set_panel_https_via_host "$domain" "$cert" "$key"; then
      log_info "面板已配置: 监听域名 ${domain},证书 ${cert} / ${key}(通过面板 API 写入)"
      return 0
    fi
  fi
  log_warn "API 设置失败(默认密码可能已被修改,或容器无法安装 curl),请登录面板手动设置: 设置 → 常规 → 监听域名 / 证书路径"
  return 1
}

# compose 方式:docker exec 进容器,用 curl 调本地面板 API
# 注意:PUT /api/system/settings 是补丁合并语义(只更新传入字段),直接传三个字段即可。
# 不要 GET→sed→PUT 全量:gin 返回紧凑 JSON(冒号后无空格),旧 sed 模式匹配不到会静默失败,
# 导致证书路径永远写不进配置(面板一直 HTTP 模式,https 打不开)——本 bug 已修。
# busybox wget 不支持 --method=PUT(曾导致容器版写入永远失败,SQLite 里证书路径不更新),
# 因此统一先确保容器内有 curl 再发 PUT。
# 安全入口(webBasePath)适配:面板设置了 /入口 时,不带前缀的 API 请求会 302,
# 先探测 public-config 的重定向 Location 拼出带前缀的 base。
set_panel_https_via_container() {
  local domain="$1" cert="$2" key="$3"
  docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$" || return 1
  local ok
  ok=$(docker exec -i "$CONTAINER_NAME" sh <<EOF
base=http://localhost:8080
command -v curl >/dev/null 2>&1 || apk add --no-cache curl >/dev/null 2>&1 || exit 9
loc=\$(curl -sI "\$base/api/system/public-config" 2>/dev/null | tr -d '\r' | grep -i '^Location:' | head -1 | sed 's/^[Ll]ocation: //')
case "\$loc" in
  http*) base="\${loc%/api/system/public-config}" ;;
  /*) base="http://localhost:8080\${loc%/api/system/public-config}" ;;
esac
token=\$(curl -s -X POST "\$base/api/login" -H 'Content-Type: application/json' -d '{"username":"admin","password":"123456"}' | sed 's/.*"token":"\\([^\"]*\\)".*/\\1/')
[ -z "\$token" ] && exit 1
curl -s -X PUT "\$base/api/system/settings" -H 'Content-Type: application/json' -H "Authorization: Bearer \$token" -d '{"webDomain":"$domain","webCertFile":"$cert","webKeyFile":"$key"}'
EOF
)
  [[ "$ok" == *"needRestart"* ]]
}

# 探测面板安全入口(webBasePath):返回 API 根地址(含入口前缀),供宿主 curl 调用
# 注意:302 Location 可能是相对路径(/dm123/api/...),需拼上主机
get_api_base() {
  local loc
  loc=$(curl -sI "http://127.0.0.1:${DM_PORT}/api/system/public-config" 2>/dev/null | tr -d '\r' | grep -i '^Location:' | head -1 | sed 's/^[Ll]ocation: //')
  if [ -n "$loc" ]; then
    case "$loc" in
      http://*|https://*) ;;
      /*) loc="http://127.0.0.1:${DM_PORT}${loc}" ;;
      *) loc="http://127.0.0.1:${DM_PORT}/${loc}" ;;
    esac
    echo "${loc%/api/system/public-config}"
  else
    echo "http://127.0.0.1:${DM_PORT}"
  fi
}

# 二进制方式:宿主 curl 调面板 API(同样是补丁合并,直接 PUT 三个字段)
set_panel_https_via_host() {
  local domain="$1" cert="$2" key="$3"
  local base token resp
  base=$(get_api_base)
  token=$(curl -s -X POST "$base/api/login" -H 'Content-Type: application/json' -d '{"username":"admin","password":"123456"}' | sed 's/.*"token":"\([^"]*\)".*/\1/')
  [ -z "$token" ] && return 1
  resp=$(curl -s -X PUT -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
    -d "{\"webDomain\":\"$domain\",\"webCertFile\":\"$cert\",\"webKeyFile\":\"$key\"}" \
    "$base/api/system/settings")
  [[ "$resp" == *"needRestart"* ]]
}

# ================= Docker Compose 安装 =================
install_compose() {
  log_step "使用 Docker Compose 方式安装"
  install_docker_if_missing
  [ -S /var/run/docker.sock ] || die "Docker socket 不存在(/var/run/docker.sock),请确认 Docker 服务已启动"
  command -v docker compose >/dev/null 2>&1 || die "未检测到 Docker Compose 插件,请安装 docker-compose-plugin"
  generate_compose
  pull_image
  log_step "启动容器..."
  ( cd "$DM_INSTALL_DIR" && docker compose up -d ) || die "容器启动失败,请查看: docker compose -f $COMPOSE_FILE logs"
  wait_ready
  # 容器内 Docker socket 校验:首次部署时 daemon 未就绪会把 socket 挂载成目录,
  # 面板将永远连不上 Docker —— 检测到异常自动重建容器
  if ! docker exec "$CONTAINER_NAME" test -S /var/run/docker.sock 2>/dev/null; then
    log_warn "容器内 Docker socket 异常(挂载成了目录),自动重建容器..."
    ( cd "$DM_INSTALL_DIR" && docker compose down && docker compose up -d ) || die "重建失败"
    wait_ready
  fi
  docker exec "$CONTAINER_NAME" test -S /var/run/docker.sock 2>/dev/null \
    && log_info "容器内 Docker socket 正常 ✓" \
    || log_warn "容器内仍未检测到 Docker socket,请检查: systemctl status docker 与 docker logs ${CONTAINER_NAME}"
  write_mode_marker compose
}

# ================= 本地二进制安装(systemd) =================
install_binary() {
  log_step "使用本地二进制方式安装(systemd)"
  local arch
  arch=$(detect_arch)
  log_info "检测到架构: $arch"
  local bin_dir="$DM_INSTALL_DIR/bin"
  local pkg="docker-manager-go-linux-${arch}.tar.gz"
  local url="https://github.com/${GITHUB_REPO}/releases/latest/download/${pkg}"
  mkdir -p "$bin_dir" "$DM_DATA_DIR" "$DM_CERT_DIR"

  log_step "下载 $pkg ..."
  local downloaded=0
  if [ "$USE_MIRROR" -eq 1 ]; then
    for proxy in "${CN_GH_PROXIES[@]}"; do
      log_step "尝试加速: ${proxy}${url}"
      if curl -fsSL --connect-timeout 10 "${proxy}${url}" -o "$DM_INSTALL_DIR/$pkg"; then
        downloaded=1
        break
      fi
    done
  fi
  [ "$downloaded" -eq 0 ] && curl -fsSL --connect-timeout 15 "$url" -o "$DM_INSTALL_DIR/$pkg" || \
    { [ "$downloaded" -eq 1 ] || die "二进制下载失败: $url"; }

  # SHA256 校验(与面板内置 updater 一致):校验失败立即中止,
  # 绝不安装损坏或被篡改的包。校验文件同样走国内加速源。
  local sum_url="$url.sha256" sum_file="$DM_INSTALL_DIR/$pkg.sha256"
  if [ "$USE_MIRROR" -eq 1 ]; then
    for proxy in "${CN_GH_PROXIES[@]}"; do
      if curl -fsSL --connect-timeout 10 "${proxy}${sum_url}" -o "$sum_file" 2>/dev/null; then
        break
      fi
    done
  fi
  [ -s "$sum_file" ] || curl -fsSL --connect-timeout 15 "$sum_url" -o "$sum_file" || die "校验文件下载失败: $sum_url"
  local expected actual
  expected=$(awk '{print $1}' "$sum_file")
  actual=$(sha256sum "$DM_INSTALL_DIR/$pkg" | awk '{print $1}')
  rm -f "$sum_file"
  if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
    rm -f "$DM_INSTALL_DIR/$pkg"
    die "SHA256 校验失败(下载的二进制可能损坏或被篡改),已中止安装"
  fi
  log_info "SHA256 校验通过 ✓"

  tar xzf "$DM_INSTALL_DIR/$pkg" -C "$bin_dir" || die "解压失败"
  chmod +x "$bin_dir/docker-manager-go/docker-manager-go"
  ln -sf "$bin_dir/docker-manager-go/docker-manager-go" /usr/local/bin/docker-manager-go
  rm -f "$DM_INSTALL_DIR/$pkg"
  [ -x /usr/local/bin/docker-manager-go ] || die "二进制安装失败"

  # systemd service
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Docker Manager Go
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=DATA_DIR=${DM_DATA_DIR}
Environment=PORT=${DM_PORT}
Environment=DOCKER_HOST=unix:///var/run/docker.sock
ExecStart=/usr/local/bin/docker-manager-go
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
  ln -sf "$SERVICE_FILE" /etc/systemd/system/docker-manager.service
  systemctl daemon-reload
  systemctl enable docker-manager >/dev/null 2>&1
  systemctl restart docker-manager
  sleep 2
  systemctl is-active docker-manager >/dev/null 2>&1 || die "服务启动失败,请查看: journalctl -u docker-manager -n 50"
  log_info "systemd 服务已启动: docker-manager"
  wait_ready
  write_mode_marker binary
}

write_mode_marker() {
  echo "$1" > "$DM_INSTALL_DIR/.install_mode"
}

read_mode() {
  cat "$DM_INSTALL_DIR/.install_mode" 2>/dev/null || echo compose
}

wait_ready() {
  log_step "等待服务就绪..."
  for i in $(seq 1 15); do
    if curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${DM_PORT}/" 2>/dev/null | grep -qE '200|302'; then
      break
    fi
    sleep 1
  done
}

# ---------- 安装状态 ----------
is_installed() {
  [ -f "$DM_INSTALL_DIR/.install_mode" ]
}

show_status() {
  local mode
  mode=$(read_mode)
  if [ "$mode" = "compose" ]; then
    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
      local state
      state=$(docker inspect -f '{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null)
      log_info "${CONTAINER_NAME}: ${GREEN_COLOR}${state}${RES} (Compose 安装)"
      return 0
    fi
    return 1
  else
    if systemctl is-active docker-manager >/dev/null 2>&1; then
      log_info "docker-manager: ${GREEN_COLOR}运行中${RES} (二进制安装, systemd)"
      return 0
    fi
    return 1
  fi
}

# ---------- 安装 ----------
install() {
  if is_installed && [ "$DM_FORCE" != "1" ]; then
    log_warn "检测到已安装($(read_mode) 方式)"
    show_status && {
      echo -e "  面板地址: ${CYAN_COLOR}http://<服务器IP>:${DM_PORT}${RES}"
      echo -e "  默认账号: admin / 123456(首次登录后请尽快修改)"
      echo -e "  如需覆盖重装: ${CYAN_COLOR}DM_FORCE=1 bash install.sh install${RES}"
      exit 0
    }
  fi

  # 选择安装方式
  local mode="$DM_MODE"
  if [ -z "$mode" ]; then
    echo
    echo -e "  ${GREEN_COLOR}1${RES}) Docker Compose 安装(推荐,自动更新镜像)"
    echo -e "  ${GREEN_COLOR}2${RES}) 本地二进制安装(systemd,无需 Docker,自动检测架构)"
    read -r -p "请选择安装方式 [1/2]: " m
    case "$m" in
      1) mode="compose" ;;
      2) mode="binary" ;;
      *) die "无效选择" ;;
    esac
  fi

  log_step "开始安装 Docker Manager(方式: $mode)"
  if [ "$mode" = "compose" ]; then
    install_compose
  else
    install_binary
  fi

  # 域名绑定(可选)
  if confirm "是否绑定域名并配置 HTTPS 证书(acme.sh 自动申请)?"; then
    ssl_domain
  fi

  echo
  echo -e "${GREEN_COLOR}==================================================${RES}"
  echo -e "  Docker Manager 安装完成!"
  echo -e "  ${BLUE_COLOR}面板地址:${RES}      http://<服务器IP>:${DM_PORT}"
  echo -e "  ${BLUE_COLOR}默认账号:${RES}      admin / 123456"
  echo -e "  ${BLUE_COLOR}数据目录:${RES}      ${DM_DATA_DIR}"
  if [ "$mode" = "compose" ]; then
    echo -e "  ${BLUE_COLOR}镜像加速:${RES}      /etc/docker/daemon.json 已挂载,可在面板设置中配置"
  fi
  echo -e "  ${GREEN_COLOR}==================================================${RES}"
  echo -e "  首次登录后请到「设置 → 安全设置」修改默认密码"
}

# ---------- 更新 ----------
update() {
  is_installed || die "尚未安装,请先执行: bash install.sh install"
  local mode
  mode=$(read_mode)
  log_step "更新中(方式: $mode)..."
  if [ "$mode" = "compose" ]; then
    pull_image
    ( cd "$DM_INSTALL_DIR" && docker compose up -d --force-recreate --pull always ) || die "更新失败"
  else
    install_binary
  fi
  show_status
  log_info "更新完成"
}

# ---------- 卸载 ----------
uninstall() {
  is_installed || die "尚未安装"
  local mode
  mode=$(read_mode)
  echo -e "${RED_COLOR}即将卸载 Docker Manager!${RES}"
  confirm "确认卸载?(数据目录 ${DM_DATA_DIR} 会保留)" || { log_info "已取消"; exit 0; }
  if [ "$mode" = "compose" ]; then
    ( cd "$DM_INSTALL_DIR" && docker compose down ) || true
    confirm "是否同时删除 compose 文件与安装目录?(数据保留)" && rm -f "$COMPOSE_FILE"
  else
    systemctl stop docker-manager 2>/dev/null || true
    systemctl disable docker-manager 2>/dev/null || true
    rm -f /etc/systemd/system/docker-manager.service
    rm -f /usr/local/bin/docker-manager-go
    systemctl daemon-reload
    confirm "是否同时删除安装目录?(数据保留)" && rm -rf "$DM_INSTALL_DIR/bin" "$DM_INSTALL_DIR/.install_mode"
  fi
  log_info "卸载完成,数据仍在 ${DM_DATA_DIR}(彻底删除: rm -rf $DM_DATA_DIR)"
}

# ---------- 服务管理 ----------
service_action() {
  local action="$1"
  is_installed || die "尚未安装,请先执行: bash install.sh install"
  local mode
  mode=$(read_mode)
  case "$mode" in
    compose)
      ( cd "$DM_INSTALL_DIR" && docker compose "$action" )
      ;;
    binary)
      systemctl "$action" docker-manager
      ;;
  esac
  show_status
}

# ================= SSL 证书 / 域名绑定(仿 3x-ui acme.sh 流程) =================
install_acme() {
  if [ -x "$HOME/.acme.sh/acme.sh" ]; then
    log_info "acme.sh 已安装"
    return 0
  fi
  log_step "安装 acme.sh..."
  curl -s https://get.acme.sh | sh || die "acme.sh 安装失败"
  [ -x "$HOME/.acme.sh/acme.sh" ] || die "acme.sh 安装失败"
  log_info "acme.sh 安装成功"
}

install_socat() {
  command -v socat >/dev/null 2>&1 && return 0
  log_step "安装 socat..."
  local os_id
  os_id=$(. /etc/os-release && echo "$ID")
  case "$os_id" in
    debian|ubuntu|armbian)
      apt-get update >/dev/null 2>&1; apt-get install -y socat >/dev/null 2>&1 ;;
    centos|rhel|rocky|almalinux|fedora|amzn)
      command -v dnf >/dev/null 2>&1 && dnf install -y socat >/dev/null 2>&1 || yum install -y socat >/dev/null 2>&1 ;;
    alpine)
      apk add socat curl openssl >/dev/null 2>&1 ;;
    *)
      log_warn "不支持的发行版,请手动安装 socat" ;;
  esac
  command -v socat >/dev/null 2>&1 || log_warn "socat 安装失败,standalone 验证可能不可用"
}

# 域名证书申请(核心:仿 3x-ui ssl_cert_issue)
ssl_domain() {
  install_acme
  install_socat
  mkdir -p "$DM_CERT_DIR"

  local domain=""
  while true; do
    read -r -p "请输入要绑定的域名: " domain
    domain="${domain// /}"
    [ -z "$domain" ] && { log_error "域名不能为空"; continue; }
    if ! is_domain "$domain"; then
      log_error "域名格式无效: $domain"
      continue
    fi
    break
  done
  log_step "域名: $domain"
  # 仿 3x-ui:先校验 A 记录指向本机,解析不对直接终止,绝不带病申请
  check_domain_dns "$domain" || return 1
  log_step "开始申请证书(acme.sh standalone 模式,需 80 端口空闲)..."
  log_info "若申请失败,常见原因: 80 端口被占用 / 防火墙未放行 80 / DNS 解析未生效"

  local cert_dir="$DM_CERT_DIR/$domain"
  mkdir -p "$cert_dir"

  "$HOME/.acme.sh/acme.sh" --issue --standalone -d "$domain" -k ec-256 \
    --server letsencrypt || {
    log_error "证书申请失败,常见原因:"
    echo -e "  1) 域名 A 记录未指向本机 IP"
    echo -e "  2) 80 端口被占用(请先停止占用 80 的服务)"
    echo -e "  3) 防火墙未放行 80 端口"
    return 1
  }

  "$HOME/.acme.sh/acme.sh" --install-cert -d "$domain" --ecc \
    --fullchain-file "$cert_dir/fullchain.cer" \
    --key-file "$cert_dir/$domain.key" || die "证书安装失败"
  chmod 644 "$cert_dir/fullchain.cer"
  chmod 600 "$cert_dir/$domain.key"
  log_info "证书已生成: $cert_dir"

  # 配置面板:监听域名 + 证书路径
  local mode
  mode=$(read_mode)
  if [ "$mode" = "compose" ]; then
    # 容器内证书挂载在 /data/cert
    set_panel_https "$domain" "/data/cert/$domain/fullchain.cer" "/data/cert/$domain/$domain.key"
    # HTTPS 端口映射:8080 → 443(https://域名 直接访问,无需端口号)
    if [ -f "$COMPOSE_FILE" ] && grep -q "${DM_PORT}:8080" "$COMPOSE_FILE"; then
      sed -i "s|${DM_PORT}:8080|443:8080|" "$COMPOSE_FILE"
      log_info "compose 端口已改为 443:8080,访问地址: https://${domain}/"
    fi
    log_step "重建容器使 HTTPS 配置生效..."
    ( cd "$DM_INSTALL_DIR" && docker compose up -d --force-recreate ) || {
      log_error "容器重建失败,请检查: cd $DM_INSTALL_DIR && docker compose up -d --force-recreate"
      return 1
    }
    # TLS 自检:等待面板以 TLS 模式启动
    sleep 3
    if docker logs "$CONTAINER_NAME" --tail 20 2>/dev/null | grep -q "TLS enabled"; then
      log_info "面板 HTTPS 启动成功 ✓"
    else
      log_warn "未检测到 TLS enabled 日志,请检查: docker logs ${CONTAINER_NAME}"
    fi
  else
    set_panel_https "$domain" "$cert_dir/fullchain.cer" "$cert_dir/$domain.key"
    systemctl restart docker-manager >/dev/null 2>&1
  fi
  log_info "HTTPS 已启用: https://${domain}/ (监听域名、证书、端口已全部自动配置)"
}

# 查看已申请证书
ssl_list() {
  if [ -d "$DM_CERT_DIR" ]; then
    echo -e "${CYAN_COLOR}已申请的证书:${RES}"
    find "$DM_CERT_DIR" -maxdepth 1 -mindepth 1 -type d -exec basename {} \; 2>/dev/null | grep -v '^$' || echo "  (无)"
  fi
  "$HOME/.acme.sh/acme.sh" --list 2>/dev/null | grep -v '^Main_Domain' | head -10 || true
}

# 强制续期
ssl_renew() {
  "$HOME/.acme.sh/acme.sh" --renew-all -f --standalone || die "续期失败"
  log_info "续期完成,请手动重启面板使新证书生效: bash install.sh restart"
}

# 删除证书
ssl_remove() {
  ssl_list
  read -r -p "请输入要删除的域名: " domain
  domain="${domain// /}"
  [ -z "$domain" ] && return
  "$HOME/.acme.sh/acme.sh" --revoke -d "$domain" 2>/dev/null || true
  "$HOME/.acme.sh/acme.sh" --remove -d "$domain" 2>/dev/null || true
  rm -rf "$HOME/.acme.sh/${domain}" "$HOME/.acme.sh/${domain}_ecc" "$DM_CERT_DIR/$domain"
  log_info "已删除证书: $domain"
}

# SSL 管理菜单
ssl_menu() {
  while true; do
    echo
    echo -e "${CYAN_COLOR}========== SSL 证书管理(域名绑定) ==========${RES}"
    echo -e "  ${GREEN_COLOR}1${RES}) 申请域名证书(acme.sh,需域名解析到本机)"
    echo -e "  ${GREEN_COLOR}2${RES}) 查看已申请证书"
    echo -e "  ${GREEN_COLOR}3${RES}) 强制续期"
    echo -e "  ${GREEN_COLOR}4${RES}) 删除证书"
    echo -e "  ${GREEN_COLOR}0${RES}) 返回"
    echo -e "${CYAN_COLOR}============================================${RES}"
    read -r -p "请选择 [0-4]: " choice
    case "$choice" in
      1) ssl_domain ;;
      2) ssl_list ;;
      3) ssl_renew ;;
      4) ssl_remove ;;
      0) return ;;
      *) log_warn "无效选择" ;;
    esac
  done
}

# ---------- 备份 / 恢复 ----------
backup() {
  is_installed || die "尚未安装"
  mkdir -p "$BACKUP_DIR"
  local name="dm-backup-$(date +%Y%m%d_%H%M%S).tar.gz"
  log_step "备份数据目录 $DM_DATA_DIR → $BACKUP_DIR/$name"
  tar czf "$BACKUP_DIR/$name" -C "$(dirname "$DM_DATA_DIR")" "$(basename "$DM_DATA_DIR")" || die "备份失败"
  log_info "备份完成: $BACKUP_DIR/$name ($(du -h "$BACKUP_DIR/$name" | cut -f1))"
}

restore() {
  is_installed || die "尚未安装"
  mkdir -p "$BACKUP_DIR"
  local list=()
  local i=0
  for f in "$BACKUP_DIR"/dm-backup-*.tar.gz; do
    [ -f "$f" ] || continue
    i=$((i + 1))
    list+=("$f")
    echo -e "  ${GREEN_COLOR}$i${RES} - $(basename "$f")"
  done
  [ "$i" -eq 0 ] && die "未找到备份文件($BACKUP_DIR 下无 dm-backup-*.tar.gz)"
  local choice
  read -r -p "请选择要恢复的备份 [1-$i]: " choice
  [[ "$choice" =~ ^[0-9]+$ ]] && [ "$choice" -ge 1 ] && [ "$choice" -le "$i" ] || die "无效选择"
  local file="${list[$((choice - 1))]}"
  confirm "恢复前会覆盖当前数据(建议先 backup),确认恢复 $file?" || { log_info "已取消"; exit 0; }
  log_step "停止服务并恢复..."
  service_action stop >/dev/null 2>&1 || true
  rm -rf "$DM_DATA_DIR"
  mkdir -p "$DM_DATA_DIR"
  tar xzf "$file" -C "$DM_INSTALL_DIR" || die "恢复失败"
  service_action start >/dev/null 2>&1 || true
  log_info "恢复完成"
}

# ---------- 重置密码 ----------
reset_passwd() {
  is_installed || die "尚未安装"
  local mode
  mode=$(read_mode)
  echo -e "${RED_COLOR}重置管理员密码:将恢复默认 admin / 123456${RES}"
  echo -e "  (2FA/用户配置会一并重置,容器/镜像等数据不受影响)"
  confirm "确认重置?" || { log_info "已取消"; exit 0; }
  # 用户数据已迁移 SQLite:旧方案删除 users.json 不再生效。
  # 改用面板启动标记:面板启动时检测到 .reset-admin-password 会把 admin 密码
  # 重置为 123456 并删除标记(与数据库模型兼容,compose/binary 通用)。
  touch "$DM_DATA_DIR/.reset-admin-password"
  chmod 600 "$DM_DATA_DIR/.reset-admin-password"
  if [ "$mode" = "compose" ]; then
    docker restart "$CONTAINER_NAME" >/dev/null 2>&1 || die "容器重启失败,请手动执行: docker restart $CONTAINER_NAME"
  else
    systemctl restart docker-manager || die "服务重启失败,请手动执行: systemctl restart docker-manager"
  fi
  log_info "密码已重置:admin / 123456,请尽快登录修改"
}

# ---------- 安装信息 ----------
info() {
  show_status
  echo -e "  安装方式:  $(read_mode)"
  echo -e "  安装目录:  $DM_INSTALL_DIR"
  echo -e "  数据目录:  $DM_DATA_DIR"
  echo -e "  证书目录:  $DM_CERT_DIR"
  echo -e "  面板端口:  $DM_PORT"
  echo -e "  面板地址:  ${CYAN_COLOR}http://<服务器IP>:${DM_PORT}${RES}"
}

# ---------- 帮助 ----------
usage() {
  cat <<EOF
Docker Manager 一键脚本 v2.0.0

用法:
  bash install.sh                 交互菜单
  bash install.sh install         安装(可选 DM_MODE=compose|binary 指定方式;DM_FORCE=1 覆盖重装)
  bash install.sh ssl             SSL 证书管理(域名绑定,acme.sh)
  bash install.sh update          更新(面板内也已支持一键在线更新)
  bash install.sh uninstall       卸载(数据保留)
  bash install.sh start|stop|restart|status
  bash install.sh backup          备份数据
  bash install.sh restore         恢复数据
  bash install.sh reset-passwd    重置密码为 admin/123456
  bash install.sh info            查看安装信息

环境变量:
  DM_PORT=8080                    面板端口
  DM_DATA_DIR=/opt/docker-manager/data   数据目录
  DM_INSTALL_DIR=/opt/docker-manager     安装目录
  DM_IMAGE=zhaoweiwen123/docker-manager-go:latest   镜像(compose)
  DM_MODE=compose|binary          安装方式
  DM_PRIVILEGED=false             特权模式(仅 compose)
EOF
}

# ---------- 交互菜单 ----------
menu() {
  while true; do
    echo
    echo -e "${CYAN_COLOR}========== Docker Manager 管理菜单 ==========${RES}"
    echo -e "  ${GREEN_COLOR}1${RES}) 安装"
    echo -e "  ${GREEN_COLOR}2${RES}) 更新"
    echo -e "  ${GREEN_COLOR}3${RES}) 卸载"
    echo -e "  ${GREEN_COLOR}4${RES}) 启动 / 停止 / 重启 / 状态"
    echo -e "  ${GREEN_COLOR}5${RES}) 备份数据"
    echo -e "  ${GREEN_COLOR}6${RES}) 恢复数据"
    echo -e "  ${GREEN_COLOR}7${RES}) 重置密码"
    echo -e "  ${GREEN_COLOR}8${RES}) SSL 证书 / 域名绑定"
    echo -e "  ${GREEN_COLOR}9${RES}) 查看信息"
    echo -e "  ${GREEN_COLOR}0${RES}) 退出"
    echo -e "${CYAN_COLOR}============================================${RES}"
    read -r -p "请选择操作 [0-9]: " choice
    case "$choice" in
      1) install ;;
      2) update ;;
      3) uninstall ;;
      4)
        read -r -p "操作(start/stop/restart/status): " act
        service_action "$act"
        ;;
      5) backup ;;
      6) restore ;;
      7) reset_passwd ;;
      8) ssl_menu ;;
      9) info ;;
      0) exit 0 ;;
      *) log_warn "无效选择" ;;
    esac
  done
}

# ---------- 入口 ----------
check_env
detect_network

case "${1:-}" in
  install)       install ;;
  ssl|ssl-cert)  ssl_menu ;;
  update)        update ;;
  uninstall)     uninstall ;;
  start|stop|restart|status) service_action "$1" ;;
  backup)        backup ;;
  restore)       restore ;;
  reset-passwd)  reset_passwd ;;
  info)          info ;;
  help|-h|--help) usage ;;
  *)             menu ;;
esac
