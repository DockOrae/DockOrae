#!/bin/bash
###############################################################################
#
# Docker Manager Go 一键安装/管理脚本
#
# Version: 1.0.0
# Last Updated: 2026-08-27
#
# Description:
#   面向 Docker Manager (https://github.com/MinimaxFlora/Docker_Manager_Go)
#   的 Docker Compose 一键脚本,提供安装、更新、卸载、启停、备份恢复、
#   密码重置等管理功能;自动检测国内/海外网络分流,国内使用镜像加速拉取。
#
# Requirements:
#   - Linux + Docker Engine(含 compose 插件)
#   - 安装需要 root 权限
#
# Usage:
#   ./install.sh              交互菜单
#   ./install.sh install      安装(--force 覆盖重装)
#   ./install.sh update       更新到最新镜像
#   ./install.sh uninstall    卸载(需确认)
#   ./install.sh start|stop|restart|status
#   ./install.sh backup       备份数据目录
#   ./install.sh restore      从备份恢复
#   ./install.sh reset-passwd 重置管理员密码(恢复 admin/123456,保留其他数据)
#   ./install.sh info         查看安装信息
#
# Env:
#   DM_PORT        面板端口(默认 8080)
#   DM_DATA_DIR    数据目录(默认 /opt/docker-manager/data)
#   DM_INSTALL_DIR 安装目录(默认 /opt/docker-manager)
#   DM_IMAGE       镜像(默认 zhaoweiwen123/docker-manager-go:latest)
#   DM_PRIVILEGED  特权模式 true/false(默认 false,开启后可自动重启宿主机 Docker)
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
DM_IMAGE="${DM_IMAGE:-zhaoweiwen123/docker-manager-go:latest}"
DM_PRIVILEGED="${DM_PRIVILEGED:-false}"
CONTAINER_NAME="docker-manager"
COMPOSE_FILE="$DM_INSTALL_DIR/docker-compose.yml"
BACKUP_DIR="$DM_INSTALL_DIR/backups"

# 国内镜像加速源(按顺序尝试)
CN_MIRRORS=(
  "docker.1panel.live"
  "dockerpull.org"
  "hub.rat.dev"
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

# ---------- 系统检测 ----------
check_env() {
  [ "$(uname -s)" != "Linux" ] && die "此脚本仅支持 Linux 系统"
  [ "$(id -u)" -ne 0 ] && die "请使用 root 权限运行: sudo bash install.sh"
  command -v docker >/dev/null 2>&1 || die "未检测到 Docker,请先安装: https://docs.docker.com/engine/install/"
  docker compose version >/dev/null 2>&1 || die "未检测到 Docker Compose 插件,请安装 docker-compose-plugin"
  docker info >/dev/null 2>&1 || die "Docker 服务未运行或无权限访问(docker ps 测试一下)"
}

# ---------- 网络检测(国内/海外分流) ----------
detect_network() {
  local CN=0
  if timeout 3 curl -sI https://www.google.com >/dev/null 2>&1; then
    CN=0
  elif timeout 3 curl -sI https://www.baidu.com >/dev/null 2>&1; then
    CN=1
  else
    # 无法判断,测 docker hub
    if timeout 5 curl -sI https://registry-1.docker.io/v2/ >/dev/null 2>&1; then
      CN=0
    else
      CN=1
    fi
  fi
  if [ "$CN" -eq 1 ]; then
    log_info "检测到网络环境: ${YELLOW_COLOR}国内${RES},拉取镜像时将使用加速源"
    USE_MIRROR=1
  else
    log_info "检测到网络环境: ${GREEN_COLOR}海外/直连${RES},将直接拉取镜像"
    USE_MIRROR=0
  fi
}

# ---------- 镜像拉取(国内走加速源 + tag) ----------
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
      docker pull "$image" || die "镜像拉取失败,请检查网络或手动执行: docker pull $image"
    }
  else
    docker pull "$image" || die "镜像拉取失败,请检查网络或手动执行: docker pull $image"
  fi
}

# ---------- compose 文件生成 ----------
generate_compose() {
  mkdir -p "$DM_INSTALL_DIR" "$DM_DATA_DIR"
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
      - /:/host:ro
      - /etc/docker:/host/etc/docker:ro
    privileged: ${DM_PRIVILEGED}
EOF
  log_info "compose 文件已生成(端口 ${DM_PORT},数据目录 ${DM_DATA_DIR})"
}

# ---------- 安装状态 ----------
is_installed() {
  [ -f "$COMPOSE_FILE" ]
}

show_status() {
  if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    local state
    state=$(docker inspect -f '{{.State.Status}}' "$CONTAINER_NAME" 2>/dev/null)
    local version
    version=$(docker inspect -f '{{.Config.Image}}' "$CONTAINER_NAME" 2>/dev/null | sed 's/.*://')
    log_info "${CONTAINER_NAME}: ${GREEN_COLOR}${state}${RES} (镜像版本 ${version:-unknown})"
    return 0
  else
    return 1
  fi
}

# ---------- 安装 ----------
install() {
  if is_installed && [ "$DM_FORCE" != "1" ]; then
    log_warn "检测到已安装(compose 文件存在: $COMPOSE_FILE)"
    if show_status; then
      echo -e "  面板地址: ${CYAN_COLOR}http://<服务器IP>:${DM_PORT}${RES}"
      echo -e "  默认账号: admin / 123456(首次登录后请尽快修改)"
      echo -e "  如需覆盖重装,请执行: ${CYAN_COLOR}DM_FORCE=1 bash install.sh install${RES}"
      exit 0
    fi
  fi

  log_step "开始安装 Docker Manager"
  generate_compose
  pull_image
  log_step "启动容器..."
  ( cd "$DM_INSTALL_DIR" && docker compose up -d ) || die "容器启动失败,请查看: docker compose -f $COMPOSE_FILE logs"
  log_step "等待服务就绪..."
  for i in $(seq 1 15); do
    if curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${DM_PORT}/" 2>/dev/null | grep -qE '200|302'; then
      break
    fi
    sleep 1
  done
  echo
  echo -e "${GREEN_COLOR}==================================================${RES}"
  echo -e "  Docker Manager 安装完成!"
  echo -e "  ${BLUE_COLOR}面板地址:${RES}      http://<服务器IP>:${DM_PORT}"
  echo -e "  ${BLUE_COLOR}默认账号:${RES}      admin / 123456"
  echo -e "  ${BLUE_COLOR}数据目录:${RES}      ${DM_DATA_DIR}"
  echo -e "  ${BLUE_COLOR}镜像加速:${RES}      /etc/docker/daemon.json 已挂载,可在面板设置中配置"
  echo -e "  ${BLUE_COLOR}宿主机终端:${RES}    /:/host 已挂载,终端页可直接进入宿主机 shell"
  echo -e "  ${GREEN_COLOR}==================================================${RES}"
  echo -e "  首次登录后请到「设置 → 安全设置」修改默认密码"
}

# ---------- 更新 ----------
update() {
  is_installed || die "尚未安装,请先执行: bash install.sh install"
  log_step "更新镜像 $DM_IMAGE ..."
  pull_image
  log_step "重建容器..."
  ( cd "$DM_INSTALL_DIR" && docker compose up -d --force-recreate --pull always ) || die "更新失败"
  show_status
  log_info "更新完成"
}

# ---------- 卸载 ----------
uninstall() {
  is_installed || die "尚未安装"
  echo -e "${RED_COLOR}即将卸载 Docker Manager!${RES}"
  confirm "确认卸载?(数据目录 ${DM_DATA_DIR} 会保留,不会删除)" || { log_info "已取消"; exit 0; }
  ( cd "$DM_INSTALL_DIR" && docker compose down ) || true
  confirm "是否同时删除 compose 文件与安装目录 ${DM_INSTALL_DIR}?(数据目录保留)" && {
    rm -f "$COMPOSE_FILE"
    log_info "compose 文件已删除"
  }
  log_info "卸载完成,数据仍在 ${DM_DATA_DIR}(如需彻底删除请手动执行: rm -rf $DM_DATA_DIR)"
}

# ---------- 服务管理 ----------
service_action() {
  local action="$1"
  is_installed || die "尚未安装,请先执行: bash install.sh install"
  ( cd "$DM_INSTALL_DIR" && docker compose "$action" )
  show_status
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
  log_step "停止容器并恢复..."
  ( cd "$DM_INSTALL_DIR" && docker compose down ) || true
  rm -rf "$DM_DATA_DIR"
  mkdir -p "$DM_DATA_DIR"
  tar xzf "$file" -C "$DM_INSTALL_DIR" || die "恢复失败"
  ( cd "$DM_INSTALL_DIR" && docker compose up -d ) || die "容器启动失败"
  log_info "恢复完成"
}

# ---------- 重置密码 ----------
reset_passwd() {
  is_installed || die "尚未安装"
  echo -e "${RED_COLOR}重置管理员密码:将删除 users.json 恢复默认 admin / 123456${RES}"
  echo -e "  (头像/2FA 等用户配置会一并重置,容器/镜像等数据不受影响)"
  confirm "确认重置?" || { log_info "已取消"; exit 0; }
  docker exec "$CONTAINER_NAME" rm -f /data/users.json 2>/dev/null || rm -f "$DM_DATA_DIR/users.json"
  docker restart "$CONTAINER_NAME" >/dev/null 2>&1
  log_info "密码已重置:admin / 123456,请尽快登录修改"
}

# ---------- 安装信息 ----------
info() {
  show_status
  echo -e "  安装目录:  $DM_INSTALL_DIR"
  echo -e "  数据目录:  $DM_DATA_DIR"
  echo -e "  面板端口:  $DM_PORT"
  echo -e "  面板地址:  ${CYAN_COLOR}http://<服务器IP>:${DM_PORT}${RES}"
}

# ---------- 帮助 ----------
usage() {
  cat <<EOF
Docker Manager 一键脚本 v1.0.0

用法:
  bash install.sh                交互菜单
  bash install.sh install        安装(--force 覆盖重装: DM_FORCE=1)
  bash install.sh update         更新镜像
  bash install.sh uninstall      卸载(数据保留)
  bash install.sh start|stop|restart|status
  bash install.sh backup         备份数据
  bash install.sh restore        恢复数据
  bash install.sh reset-passwd   重置密码为 admin/123456
  bash install.sh info           查看安装信息

环境变量:
  DM_PORT=8080                   面板端口
  DM_DATA_DIR=/opt/docker-manager/data   数据目录
  DM_INSTALL_DIR=/opt/docker-manager     安装目录
  DM_IMAGE=zhaoweiwen123/docker-manager-go:latest   镜像
  DM_PRIVILEGED=false            特权模式(开启后可自动重启宿主机 Docker)
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
    echo -e "  ${GREEN_COLOR}8${RES}) 查看信息"
    echo -e "  ${GREEN_COLOR}0${RES}) 退出"
    echo -e "${CYAN_COLOR}============================================${RES}"
    read -r -p "请选择操作 [0-8]: " choice
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
      8) info ;;
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
