[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/docker-manager-dark.svg">
    <img alt="Docker Manager Go" src="./media/docker-manager-light.svg" width="850">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/releases"><img src="https://img.shields.io/github/v/release/MinimaxFlora/Docker_Manager_Go" alt="Release"></a>
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/actions"><img src="https://img.shields.io/github/actions/workflow/status/MinimaxFlora/Docker_Manager_Go/release.yml.svg" alt="Build"></a>
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/MinimaxFlora/Docker_Manager_Go.svg" alt="Go Version"></a>
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest"><img src="https://img.shields.io/github/downloads/MinimaxFlora/Docker_Manager_Go/total.svg" alt="Downloads"></a>
  <a href="https://hub.docker.com/r/zhaoweiwen123/docker-manager-go"><img src="https://img.shields.io/docker/pulls/zhaoweiwen123/docker-manager-go.svg" alt="Docker Pulls"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
</p>

**Docker Manager Go** 是一款使用 **Go** 语言编写的现代化、美观的 Docker 管理面板([gin](https://github.com/gin-gonic/gin) + 官方 [Moby Docker SDK](https://github.com/moby/moby)),前端采用 **Vue 3**。界面交互设计参考 1Panel,支持深色/浅色主题与粉色品牌色,系统状态页参照 3x-ui 设计。

> [!IMPORTANT]
> 本项目仅限个人使用。请勿将其用于非法用途,或在未经适当授权的情况下用于生产环境。

## 功能特性

- **容器管理** — 创建 / 启动 / 停止 / 重启 / 暂停 / 删除 / 检查 / 附加,内置 **容器终端**(WebSocket)。
- **镜像管理** — 实时进度拉取、删除、清理未使用的镜像。
- **网络管理** — 创建 / 删除 / 检查(子网与网关配置)。
- **存储卷管理** — 创建(本地 / NFS)/ 删除 / 检查。
- **应用商店** — 260+ 个一键安装应用(数据源对齐 1Panel 应用商店:图标 / 参数表单 / 多版本);首次启动自动同步(无需手动操作)、一键安装 / 升级,带「可升级」徽标。
- **Compose 堆栈管理** — YAML 编辑器、一键部署(流式输出)、启动/停止与拆除。
- **实时监控** — 3x-ui 风格状态页:CPU / 内存 / 交换分区 / 存储卡片(带迷你走势图)、网络吞吐与磁盘 I/O 曲线、容器/镜像/存储卷数量、面板进程统计,以及可切换可见性的公网 IP。
- **许可证** — 在线授权(由 Docker_Manager_License 签发:Ed25519 签名 Key / 设备绑定 / 每 24h 周期验证 / 7 天宽限期 / 吊销即时生效);存量用户保留离线激活方式。
- **镜像加速** — 直接在面板中配置 `daemon.json` 镜像加速源。
- **多语言** — 14 种界面语言,支持深色与浅色主题。
- **安全** — TOTP 双因素认证、JWT 会话、头像上传。
- **事件流** — 实时 Docker 事件推送至仪表盘。

- **面板设置(仿 1Panel)** — 安全入口(设置后仅可通过 /入口 访问面板)、未认证响应码(200 帮助页/400/401/403/404/408/416/444/500)、面板监听域名白名单(绑定域名后 IP 访问关闭)、面板 SSL 证书路径、密码过期与复杂度策略、出站代理服务器。
- **工具箱** — 设备信息、Docker 磁盘清理(已停止容器/未使用镜像与卷/构建缓存)、Fail2ban 登录防护(自动封禁/封禁列表/解封)。

## 🤖 Agent Skill

本仓库内置 [GitHub Agent Skill](.github/skills/docker-manager-user-guide/SKILL.md) — `docker-manager-user-guide`,为 AI 助手(Copilot / Claude / ChatGPT 等)提供面板配置、部署与排障知识库。直接问你的 AI:Docker Manager 面板的安全入口/域名绑定/工具箱怎么配置?

## 快速开始

### 一键安装(推荐)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/install.sh)
```

安装脚本会:

- 自动检测您的网络环境(国内/海外)并使用加速源。
- 如果系统未安装 Docker,**自动安装 Docker**(Debian / Ubuntu amd64 / arm64,最新稳定版)。
- 可让您选择安装方式:
  1. **Docker Compose**(推荐)— 基于镜像,易于更新。
  2. **本地二进制文件**(systemd)— 无需 Docker;自动检测架构(amd64 / arm64 / armv5-7 / 386 / s390x)。
- 可选绑定 **HTTPS 域名**(通过 acme.sh 签发 Let's Encrypt 证书)。

常用命令:

```bash
sudo bash install.sh install         # Install (DM_MODE=compose|binary to force a method, DM_FORCE=1 to reinstall)
sudo bash install.sh ssl             # SSL certificate management (domain binding)
sudo bash install.sh update          # Update
sudo bash install.sh uninstall       # Uninstall (data kept)
sudo bash install.sh start|stop|restart|status
sudo bash install.sh backup          # Backup data
sudo bash install.sh restore         # Restore data
sudo bash install.sh reset-passwd    # Reset password to admin / 123456
sudo bash install.sh info            # Show installation info
```

### Docker Compose(手动)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  zhaoweiwen123/docker-manager-go:latest
```

也可以使用随附的 `docker-compose.yml`。对于远程 Docker 主机,请设置 `DOCKER_HOST=tcp://<host>:2375`。

### 二进制文件(手动)

从 [Releases 发布页](https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest) 下载 `docker-manager-go-linux-<arch>.tar.gz`,解压后运行:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## 域名绑定(SSL)

面板支持通过 **设置 → 常规 → 证书** 中配置的证书路径启用 HTTPS。使用安装脚本的 `ssl` 菜单可通过 acme.sh 自动签发 Let's Encrypt 证书(HTTP-01 独立验证 — 请确保域名解析到本机且 80 端口空闲):

```bash
sudo bash install.sh ssl
```

证书路径会自动写入面板设置,重启后 HTTPS 生效。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DATA_DIR` | `./data` | 数据目录(SQLite 数据库、设置、用户) |
| `PORT` | `8080` | 面板监听端口 |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | Docker 守护进程地址 |
| `TZ` | — | 容器时区 |

## 支持平台

- **二进制文件**(Linux):amd64、arm64、armv5、armv6、armv7、386、s390x
- **Docker 镜像**:linux/amd64、linux/arm64、linux/arm/v7、linux/arm/v6、linux/s390x
- **面板运行环境**:Linux(生产)、Windows(开发)

## 支持的语言

English、简体中文、繁體中文、日本語、한국어、Русский、Türkçe、Español、Português (Brasil)、Tiếng Việt、Indonesia、Українська、العربية、فارسی — 共 14 种语言,自动检测并支持一键切换。

## 许可证

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
