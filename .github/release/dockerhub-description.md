# 🐳 Docker Manager Go

**Go 编写的 Docker 管理面板** — [gin](https://github.com/gin-gonic/gin) + 官方 [Moby Docker SDK](https://github.com/moby/moby) + **Vue 3** 前端。UI 参考 1Panel 交互设计,系统状态页仿 3x-ui,粉色品牌色,支持深/浅色主题,14 种语言。

## ✨ 特性

- **容器管理** — 创建 / 启动 / 停止 / 重启 / 删除 / 详情 / 内置 Web 终端
- **镜像管理** — 拉取(实时进度)/ 删除 / 清理未使用镜像
- **网络 / 卷 / Compose 栈** — 完整生命周期管理,YAML 一键部署
- **实时监控** — 3x-ui 风格状态页:CPU / 内存 / 磁盘卡片 + 曲线,容器 / 镜像 / 卷统计,公网 IP
- **终端** — 宿主机终端(chroot /host)、容器终端、SSH 主机管理(密码 / 密钥认证)
- **面板设置(仿 1Panel)** — 安全入口、未认证设置(200~500 响应码)、域名绑定(设置后 IP 访问关闭)、面板 SSL、密码过期与复杂度
- **工具箱** — 设备信息、Docker 磁盘清理、Fail2ban 登录防护
- **通知** — Telegram / 邮件事件通知
- **许可证** — 离线 Pro 许可证(文件激活 / 设备绑定)

## 🚀 快速开始

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v dm-data:/data \
  zhaoweiwen123/docker-manager-go:latest
```

默认账号 `admin / 123456`(首次登录强制修改密码)。

**一键安装脚本**(自动检测国内/海外网络、自动安装 Docker、可选域名 HTTPS):

```bash
bash <(curl -Ls https://raw.githubusercontent.com/DockerManger/Docker_Manager_Go/master/install.sh)
```

## 📖 更多

- GitHub 源码: <https://github.com/DockerManger/Docker_Manager_Go>
- 发布与更新: `bash install.sh update`
- 个人使用免费;Pro 许可证支持离线激活
<!-- 触发描述更新 -->
