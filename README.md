# Docker Manager(Go)

> **English**: [README_EN.md](README_EN.md)

一个使用 **Go**(gin + 官方 Docker SDK)编写、**Vue 3** 前端的美观 Docker 管理面板。参照 1Panel 的交互设计,深色/亮色双主题 + 粉色品牌色,支持:

- 📦 **容器管理**:创建 / 启动 / 停止 / 重启 / 暂停 / 删除 / 详情 / Web 终端
- 🖼️ **镜像管理**:拉取(实时进度)/ 删除 / 详情 / 清理未使用
- 🌐 **网络管理**:创建 / 删除 / 详情(支持子网与网关配置)
- 💾 **卷管理**:创建 / 删除 / 详情
- 🧩 **Compose 栈管理**:YAML 编辑 / 一键部署(流式输出)/ 启停 / 下架
- 📊 **实时监控**:CPU / 内存 / 负载 / 磁盘环形图表、网络流量与磁盘 IO 实时曲线
- 🖥️ **终端**:宿主机终端(chroot /host)、容器终端、**SSH 主机管理**(分组 / 连接 / 密码与密钥认证)、快速命令、终端外观配置
- 🗝️ **许可证**:专业版离线授权(文件上传激活 / 绑定设备 / 解绑),未授权时限制创建容器与 Compose 部署
- ⚡ **镜像加速**:面板内直接配置 daemon.json 的 registry-mirrors
- 🌐 **国际化**:简体中文 / English 双语,自动检测 + 一键切换
- 🔔 **事件流**:Docker 事件实时推送,仪表盘自动刷新
- 🔐 **安全**:登录双因素验证(TOTP)、JWT 会话、头像上传

> 本仓库由 [Docker_Manager_Rust](https://github.com/MinimaxFlora/Docker_Manager_Rust) 全量移植而来,后端从 Rust(axum + bollard)重写为 Go(gin + moby Docker SDK),前端与 API 契约保持不变。

## 快速开始

### 一键安装(推荐)

```bash
curl -sSL https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/install.sh -o install.sh
sudo bash install.sh
```

脚本会自动检测网络环境(国内自动使用镜像加速源拉取),生成 compose 文件并启动面板。常用命令:

```bash
sudo bash install.sh install         # 安装(已安装会提示,DM_FORCE=1 覆盖重装)
sudo bash install.sh update          # 更新到最新镜像
sudo bash install.sh uninstall       # 卸载(数据保留)
sudo bash install.sh status          # 查看状态
sudo bash install.sh backup          # 备份数据
sudo bash install.sh restore         # 恢复数据
sudo bash install.sh reset-passwd    # 重置密码为 admin / 123456
```

环境变量:

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DM_PORT` | `8080` | 面板端口 |
| `DM_DATA_DIR` | `/opt/docker-manager/data` | 数据目录 |
| `DM_INSTALL_DIR` | `/opt/docker-manager` | 安装目录(compose 文件位置) |
| `DM_IMAGE` | `zhaoweiwen123/docker-manager-go:latest` | 镜像 |
| `DM_PRIVILEGED` | `false` | 特权模式(开启后可在面板自动重启宿主机 Docker) |

### Docker Compose 手动部署

```bash
git clone https://github.com/MinimaxFlora/Docker_Manager_Go.git
cd Docker_Manager_Go
docker compose up -d
```

面板地址:`http://<服务器IP>:8080`,默认账号 **admin / 123456**(首次登录后请修改密码)。

## 本地开发

```bash
# 前端(需要 Node 18+)
cd web
npm install --registry=https://registry.npmmirror.com
npm run dev          # Vite dev server

# 后端(需要 Go 1.26+)
go build -o docker-manager-go .   # 依赖 web/dist 已构建(go:embed)
DATA_DIR=./data PORT=8080 ./docker-manager-go
```

后端默认连接:
- Linux:unix socket `/var/run/docker.sock`(可用 `DOCKER_HOST` 覆盖)
- Windows:TCP `127.0.0.1:2375`(无 Docker 也能启动面板,API 返回 502 提示)

## 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.26、[gin](https://github.com/gin-gonic/gin)、[moby Docker SDK](https://github.com/moby/moby)、gorilla/websocket、x/crypto/ssh、argon2id、TOTP |
| 前端 | Vue 3、Vite、Tailwind CSS 4、xterm.js、vue-i18n |
| 部署 | 静态编译单二进制 + `go:embed` 内嵌前端,Alpine 基础镜像 |

## 许可证

MIT
