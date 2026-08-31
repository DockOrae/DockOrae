<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/docker-manager-dark.svg">
    <img alt="DockOrae" src="./media/docker-manager-light.svg" width="850">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/DockOrae/DockOrae/releases"><img src="https://img.shields.io/github/v/release/DockOrae/DockOrae" alt="Release"></a>
  <a href="https://github.com/DockOrae/DockOrae/actions"><img src="https://img.shields.io/github/actions/workflow/status/DockOrae/DockOrae/release.yml.svg" alt="Build"></a>
  <a href="https://github.com/DockOrae/DockOrae/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/DockOrae/DockOrae.svg" alt="Go Version"></a>
  <a href="https://github.com/DockOrae/DockOrae/releases/latest"><img src="https://img.shields.io/github/downloads/DockOrae/DockOrae/total.svg" alt="Downloads"></a>
  <a href="https://hub.docker.com/r/dockorae/dockorae"><img src="https://img.shields.io/docker/pulls/dockorae/dockorae.svg" alt="Docker Pulls"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
</p>

**DockOrae** 是一款使用 **Go** 语言编写的现代化、美观的 Docker 管理面板([gin](https://github.com/gin-gonic/gin) + 官方 [Moby Docker SDK](https://github.com/moby/moby)),前端采用 **Vue 3**。界面交互设计参考 1Panel,支持深色 / 浅色主题与粉色品牌色,系统状态页参照 3x-ui 设计。

> [!IMPORTANT]
> 本项目仅限个人使用。请勿将其用于非法用途,或在未经适当授权的情况下用于生产环境。

## 功能特性

- **容器管理** — 创建 / 启动 / 停止 / 重启 / 暂停 / 删除 / 检查 / 附加,内置 **容器终端**(WebSocket)。
- **镜像管理** — 实时进度拉取、删除、清理未使用的镜像。
- **网络管理** — 创建 / 删除 / 检查(子网与网关配置)。
- **存储卷管理** — 创建(**本地 / NFS**)/ 删除 / 检查。
- **应用商店** — 260+ 个一键安装应用(数据源对齐 1Panel 应用商店:图标 / 参数表单 / 多版本);首次启动自动同步(无需手动操作),一键安装 / 升级,带「可升级」徽标。
- **Compose 堆栈管理** — YAML 编辑器、一键部署(流式输出)、启动 / 停止与拆除。
- **实时监控** — 3x-ui 风格状态页:CPU / 内存 / 交换分区 / 存储卡片(带迷你走势图)、网络吞吐与磁盘 I/O 曲线、容器 / 镜像 / 存储卷数量、面板进程统计,以及可切换可见性的公网 IP。
- **许可证** — 在线授权(由 Docker_Manager_License 签发:Ed25519 签名 Key / 设备绑定 / 每 24h 周期验证 / 7 天宽限期 / 吊销即时生效);存量用户保留离线激活方式。
- **镜像加速** — 直接在面板中配置 `daemon.json` 镜像加速源。
- **多语言** — 14 种界面语言,支持深色与浅色主题。
- **安全** — TOTP 双因素认证、JWT 会话、头像上传。
- **面板设置(仿 1Panel)** — 安全入口(设置后仅可通过 `/入口` 访问面板)、未认证响应码(200 帮助页 / 400 / 401 / 403 / 404 / 408 / 416 / 444 / 500)、面板监听域名白名单(绑定域名后 IP 访问关闭)、面板 SSL 证书路径、密码过期与复杂度策略、出站代理服务器。
- **工具箱** — 设备信息、Docker 磁盘清理(已停止容器 / 未使用镜像与卷 / 构建缓存)、Fail2ban 登录防护(自动封禁 / 封禁列表 / 解封)。
- **事件流** — 实时 Docker 事件推送至仪表盘。
- **在线更新** — 自动检查 GitHub Releases(底部版本徽标提示新版本),一键更新,支持两种部署方式:compose(独立辅助容器重新拉取并重建面板)或二进制(原子自替换 + systemd 重启)。

## 🤖 Agent Skill

本仓库内置 [GitHub Agent Skill](.github/skills/docker-manager-user-guide/SKILL.md) — `docker-manager-user-guide` — 面向 AI 助手(Copilot / Claude / ChatGPT 等)的知识库,涵盖面板配置、部署与故障排查。直接问你的 AI:*"如何配置 Docker Manager 面板的安全入口 / 域名绑定 / 工具箱?"*

## 快速开始

### 一键安装(推荐)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/DockOrae/DockOrae/master/install.sh)
```

安装脚本特性:

- 自动检测网络(国内 / 海外)并使用加速源。
- 缺少 Docker 时**自动安装**(Debian / Ubuntu amd64 / arm64,最新稳定版)。
- 可选安装方式:
  1. **Docker Compose**(推荐)— 基于镜像,更新方便。
  2. **本地二进制**(systemd)— 无需 Docker;自动检测架构(amd64 / arm64 / armv5-7 / 386 / s390x)。
- 可选**域名 + HTTPS 绑定**(acme.sh 签发 Let's Encrypt 证书)。

常用命令:

```bash
sudo bash install.sh install         # 安装(DM_MODE=compose|binary 强制方式,DM_FORCE=1 强制重装)
sudo bash install.sh ssl             # SSL 证书管理(域名绑定)
sudo bash install.sh update          # 更新
sudo bash install.sh uninstall       # 卸载(保留数据)
sudo bash install.sh start|stop|restart|status
sudo bash install.sh backup          # 备份数据
sudo bash install.sh restore         # 恢复数据
sudo bash install.sh reset-passwd    # 重置密码为 admin / 123456
sudo bash install.sh info            # 查看安装信息
```

### Docker Compose(手动)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  dockorae/dockorae:latest
```

或使用仓库自带的 `docker-compose.yml`。远程 Docker 主机请设置 `DOCKER_HOST=tcp://<host>:2375`。

### 二进制(手动)

从 [Releases 页面](https://github.com/DockOrae/DockOrae/releases/latest) 下载 `docker-manager-go-linux-<arch>.tar.gz`,解压后运行:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## 域名绑定(SSL)

面板支持在 **设置 → 常规 → 证书** 中配置证书路径启用 HTTPS。也可使用安装脚本的 `ssl` 菜单通过 acme.sh 自动签发 Let's Encrypt 证书(HTTP-01 独立验证 — 请确保域名解析到本机且 80 端口空闲):

```bash
sudo bash install.sh ssl
```

证书路径会自动写入面板设置,重启后 HTTPS 生效。

## 在线授权(Pro)

Pro 功能(Compose 堆栈、容器创建、应用商店安装)由在线许可证门控:面板针对 License Server 验证 Ed25519 签名的 License Key,包含设备绑定、每 24h 周期验证、7 天宽限期与吊销即时生效。

安全模型(V3):

- **License Key** 仅用于首次激活 / 重新激活(本地 Ed25519 验证)
- 运行期验证使用 **Activation Token**(本地存储于 `license.json`,权限 0600;服务器仅保存 SHA-256 哈希 — 绝不保存明文)
- 每次 verify / deactivate 携带 `timestamp + nonce`(重放保护)
- 服务器每次响应返回 `server_time`;面板维护 `clock_offset`(可信时间)并检测本地**时钟回拨**(>5min → 禁用 Pro)
- 服务端版本控制:`minimum_client_version`(升级提示)与 `blocked_versions`(紧急封禁 → 禁用 Pro)

### 1. 部署 License Server

在任何服务器部署 [Docker_Manager_License](https://github.com/DockOrae/Docker_Manager_License) — 单容器、80 端口、零配置。**直连 IP** 与 **域名 + Cloudflare HTTPS** 两种场景的分步指南见 **[docs/DEPLOY.md](https://github.com/DockOrae/Docker_Manager_License/blob/master/docs/DEPLOY.md)**。

> ⚠️ 请使用与本面板内置公钥配对的私钥(License 仓库中的 `private/license.key`)— 否则面板签名验证会失败。参见部署指南第 3 步。

### 2. 将面板指向你的 License Server

默认使用官方服务器 `https://manager.kejizero.xyz/license-api`(无需配置)。自建 License Server 时设置环境变量:

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DM_LICENSE_SERVER_URL` | `https://manager.kejizero.xyz/license-api` | License Server 基础地址,如 `http://<ip>/license-api` 或 `https://license.example.com/license-api`。空字符串 = 离线模式(仅旧版 Key)。 |

### 3. 激活

面板 → **设置 → 许可证** → **添加** → 粘贴由 License 管理面板签发的 License Key → **激活**。状态徽标显示在线验证状态;点击 **立即验证** 可即时生效吊销(否则最长 24h)。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DATA_DIR` | `./data` | 数据目录(SQLite 数据库、设置、用户) |
| `PORT` | `8080` | 面板监听端口 |
| `DOCKER_HOST` | `unix:///var/run/docker.sock`(Linux) | Docker daemon 地址 |
| `TZ` | — | 容器时区 |

## 支持平台

- **二进制**(Linux):amd64、arm64、armv5、armv6、armv7、386、s390x
- **Docker 镜像**:linux/amd64、linux/arm64、linux/arm/v7、linux/arm/v6、linux/s390x
- **面板运行**:Linux(生产)、Windows(开发)

## 支持语言

English、简体中文、繁體中文、日本語、한국어、Русский、Türkçe、Español、Português (Brasil)、Tiếng Việt、Indonesia、Українська、العربية、فارسی — 共 14 种语言,自动检测并一键切换。

## License

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
