---
name: docker-manager-user-guide
description: 'Docker Manager Go 用户功能指南。用于回答用户关于 Docker Manager 面板如何配置/使用/排障的问题，包括：安装部署、域名 HTTPS、面板设置各 tab 配置项（安全入口/未认证设置/面板监听域名/强制 HTTPS/密码策略/TG/邮件通知）、容器与镜像管理、应用商店（一键安装/同步/可升级）、在线更新、Compose 栈、许可证等。每个配置项均标注了对应的 Settings JSON 字段、生效条件（是否需重启面板）。Use when user asks how to configure, use, or troubleshoot any Docker Manager Go panel feature.'
instructions: |
  You are a Docker Manager Go expert assistant. Docker Manager Go is a Docker management panel written in Go (gin + official Moby Docker SDK) with a Vue 3 frontend, developed by MinimaxFlora (github.com/DockOrae/DockOrae). UI is inspired by 1Panel interaction design and 3x-ui status page, with dark/light themes and a pink (#ec4899) brand color.

  When answering user questions about Docker Manager Go:
  1. When users report access problems (panel unreachable, https 打不开, cannot log in), FIRST check in order: DNS A record points to the VPS public IP (not a private/reserved segment — there was an incident where 28.0.1.x got filled in), the panel container status (docker compose ps / logs), and the certificate paths written into SQLite settings (webCertFile/webKeyFile must point to real files inside the container, e.g. /data/cert/...).
  2. Give precise CLI commands for the VPS (systemctl/docker compose/curl) and ask the user to paste back the output. For panel API checks use curl http://<ip>:8080/api/health and /api/system/public-config.
  3. Once the root cause is identified, provide web UI navigation paths (e.g. 面板设置 → 常规 → 安全入口) to fix the configuration. Remember: webPort, webBasePath (安全入口), webListen changes require a panel RESTART to take effect (Router is built at startup); webDomain, noAuthSetting, sessionMaxAge take effect immediately.
  4. For feature configuration questions (how to enable/disable/set options), provide UI paths directly — no debug needed.
  5. Explain underlying principles (Host-header validation for webDomain, gin Router built at startup so basePath needs restart, noAuthSetting response codes) — not just steps.
  6. Never guess — if information is not covered in this document, consult the source code (https://github.com/DockOrae/DockOrae) — internal/settings/settings.go for all Settings fields, internal/api/ for handlers, install.sh for the installer, frontend repo (https://github.com/DockOrae/DockOrae-Frontend) web/src/views/SettingsView.vue for the settings UI. For Docker-specific questions refer to Docker/Moby SDK docs.
  7. Cite sources when information comes from external queries.

  IMPORTANT deployment facts:
  - Default panel port 8080 (configurable via PORT env or webPort setting); default account admin / 123456, must change password on first login.
  - Data dir defaults to ./data (env DATA_DIR); SQLite DB stores settings; settings are patched via PUT /api/system/settings (merge semantics — only send changed fields; secret fields tgBotToken/smtpPass/proxyPass return masked and are NOT overwritten when left empty).
  - install.sh supports compose/standalone installs, domain binding with acme.sh certificates (host /opt/docker-manager/cert/ → container /data/cert/), and validates the domain A record against the VPS public IP before issuing certs (check_domain_dns). DM_PUBLIC_IP=<ip> overrides public IP detection.
  - 面板监听域名 (webDomain) acts as a Host whitelist: when set, the panel is only reachable via that domain (other Hosts get 404); localhost/127.0.0.1/::1 are always allowed for local ops.
  - 安全入口 (webBasePath, e.g. /dm123): when set, the panel is only reachable via /dm123/... — other paths 302 to the entrance; requires panel restart; static /assets and /logo.svg are always served.
  - 未认证设置 (noAuthSetting): response code when accessing API without login — 200 help page / 400 / 401 / 403 / 404 / 408 / 416 / 444 (connection closed) / 500, default 401.
  - Login failure events show up in 面板设置 → 日志; 401 响应携带登录失败提示。
  - 应用商店 (App Store): 数据源为 DockOrae/DockOrae-Apps 仓库(264 个应用,1Panel 同款结构 data.yml + docker-compose.yml + formFields 参数 + logo.png)。**启动时后台自动同步一次**(检测数据目录缺失才拉取,幂等),全新部署打开应用商店即有数据,无需手动操作;右上角「同步应用商店」按钮用于手动更新。同步可用环境变量 DM_APPSTORE_REPO/DM_APPSTORE_URL 覆盖仓库/下载地址。安装流程:选版本 → 填参数表单(1Panel formFields)→ 自动创建 1panel-network 外部网络 → compose up。已安装且版本非最新时卡片显示「可升级」黄色徽标,一键升级会重渲染最新版 compose。
  - 资源可见性 (1Panel 同款): 容器/Compose 列表只显示面板管理的资源 — 有 createdBy 标签(面板创建=createdBy docker-manager,应用商店=createdBy Apps)或 compose 项目在面板数据目录(面板接管/编排)。宿主机直接部署的外部容器与 compose 不显示;若宿主机资源需在面板管理,用面板"接管"粘贴 compose 即可。

type: knowledge-base
tags: [docker, docker-manager, panel, go, gin, vue, container, devops]
argument-hint: '询问 Docker Manager 面板功能如何配置/使用/排障'
user-invocable: true
disable-model-invocation: false
---

# Docker Manager Go 完整功能参考指南

## AI 行为总则

> **本文档是 AI 的知识库。任何 AI 模型在回答 Docker Manager Go 相关问题时均应遵循以下原则。**

> **排查优先级(从快到慢,逐层递进)**:
> ① **先查网络可达性** — 用户报告"访问不了":先确认 DNS A 记录是否指向 VPS 公网 IP(曾有 28.0.1.x 保留段事故,域名解析到错误 IP 导致永远打不开)、VPS 防火墙端口(80/443/8080)是否放行;
> ② **再查容器状态** — `docker compose ps`(compose 安装)或 `systemctl status docker-manager`(二进制安装),容器反复重启看日志 `docker compose logs --tail 50`;
> ③ **最后查配置** — 证书路径是否真实存在于容器内、webBasePath/安全入口是否设置后未重启、webDomain 域名白名单是否把当前访问方式挡了(可先用 localhost 回环验证是否被白名单拦截)。

---

## 第一部分:项目概述

- **定位**:Go 编写的 Docker 管理面板(单二进制,前端 Vue3 内嵌),UI 参考 1Panel 交互 + 3x-ui 系统状态页,粉色 #ec4899 品牌色,支持深/浅色主题,14 种语言。
- **架构**:`main.go`(入口)+ `web.go`(go:embed public/dist 静态资源)+ `internal/api`(gin 路由与 handler)+ `internal/settings`(SQLite 设置存储)+ `internal/state`(内存状态)+ `internal/auth`(JWT/TOTP)+ `internal/db`(SQLite)+ `internal/netutil`(代理感知 HTTP 客户端)。
- **存储**:SQLite(`data/` 目录,`DATA_DIR` 环境变量可覆盖)。设置存 `settings` 表 `main` 键(JSON)。
- **默认账号**:`admin / 123456`(首次登录强制改密)。
- **默认端口**:`8080`(`PORT` 环境变量可覆盖)。
- **镜像**:Docker Hub `zhaoweiwen123/dockorae`(GitHub Actions 构建,只推 latest 标签)。
- **仓库**:github.com/DockOrae/DockOrae(master 分支;发布含 ipk/apk 的仓库结构不同,本指南仅针对面板本体)。

---

## 第二部分:功能与 UI 导航

### 系统状态(/)
- CPU/内存/交换/存储卡片 + 迷你趋势图;网络吞吐/磁盘 I/O 曲线;容器/镜像/卷计数;面板进程统计;公网 IP(可切换显示/隐藏)。
- 布局仿 3x-ui 系统状态页。

### 容器管理(容器)
- 列表(全部/运行/停止)、创建(表单,含端口/环境变量/卷/网络)、启动/停止/重启/暂停/删除/查看详情/进入终端(Web Terminal)。
- 详情页:日志(跟随)、Stats、进程、变更、配置(JSON)、启动参数。

### 镜像管理(镜像)
- 列表、拉取(实时进度,支持从 Dockerfile 构建)、删除、清理未使用镜像、查看详情(标签/大小/层)。

### 网络管理(网络)
- 列表、创建(子网/网关配置)、删除、查看详情(连接的容器)。

### 卷管理(卷)
- 列表、创建、删除、查看详情。

### Compose 栈(Compose)
- 项目列表、YAML 编辑器、一键部署(流式输出)、启动/停止/删除、查看详情。

### 面板设置(面板设置)
- 子菜单:**常规 / 安全 / Telegram / 邮件 / 许可证 / 关于**;常规页内横向 tab:**常规 / 证书 / 日期和时间**;安全页内横向 tab:**管理员凭证 / 双因素验证**。

### 在线更新(页脚版本图标)
- 面板启动后每 10 分钟静默检查 GitHub Releases(`api.github.com/repos/DockOrae/DockOrae/releases/latest`,结果缓存 10 分钟,**失败不缓存**;`DM_UPDATE_API` 环境变量可覆盖检测接口,测试用)。
- 有新版时页脚版本号位置的**下载图标亮粉色红点**(`.update-dot`),点击弹出更新详情 Modal:当前/最新版本对比、发布时间、release notes、GitHub 链接、立即更新按钮(确认后执行)。
- **一键更新按部署模式自动分流**(`deploymentMode()`;`DM_DEPLOY_MODE=compose|binary` 可强制,生产自动判断:cgroup 含 docker 容器 ID → compose,否则 binary):
  - **compose 部署(容器内)**:探测宿主 docker-compose.yml(优先从自身容器的 `/data` 挂载反推宿主安装目录,兜底 `/host/opt/docker-manager/docker-compose.yml`,返回宿主路径)→ 拉取 `docker/compose:latest` → 启动独立 `dm-update-helper` 容器(挂 docker.sock + compose 目录只读,`AutoRemove`)执行 `compose up -d --force-recreate --pull always` → 面板容器被重建,短暂断连后自动恢复。
  - **binary 部署(systemd)**:下载 GitHub Release 资产 `dockorae-linux-<amd64|arm64>.tar.gz` → 解压 → **先复制到 `/proc/self/exe` 同目录再原子 rename**(避免 /tmp tmpfs 跨文件系统 EXDEV)→ 保留 `.old` 备份 → 1.5 秒后 `systemctl restart docker-manager`。
- 更新中前端轮询 `/api/update/check` 直到 `has_update=false` 且无错误(新版本上线)才算完成。
- **排障**:检测失败显示"检查更新失败"(GitHub API 403/网络不通,不阻塞面板);compose 模式找不到 compose 文件会提示**已探测的路径**(数据目录不在安装目录下等特殊情况,需手动 `install.sh update`);binary 模式需 root 权限(能写 /usr/local/bin + systemctl);更新后版本号在 `main.go`(CI 提取)与 `internal/api/update.go` 的 `AppVersion` 两处同步。

---

## 第三部分:面板设置配置项详解

> 设置接口:`GET/PUT /api/system/settings`(PUT 为补丁合并语义,只传要改的字段;secret 字段脱敏返回,留空不覆盖)。字段名即 Settings JSON key。**修改端口/监听/安全入口需重启面板生效**;其余即时生效。

### 常规 tab

| 配置项 | JSON 字段 | 生效 | 说明 |
|--------|----------|------|------|
| 面板监听 IP | `webListen` | 重启 | 空 = 0.0.0.0 监听所有 IPv4;填 127.0.0.1 配合反代 |
| 面板监听域名 | `webDomain` | 即时 | **Host 白名单**:设置后仅该域名可访问,其他 Host(含公网 IP 直访)返回 404;localhost/127.0.0.1 放行 |
| 面板监听端口 | `webPort` | 重启 | 默认 8080 |
| 安全入口 | `webBasePath` | 重启 | 设置 /xxx 后仅 /xxx 前缀可访问,其余 302 到入口;静态资源 /assets 放行 |
| 未认证设置 | `noAuthSetting` | 即时 | 未登录访问 API 的响应码:200 帮助页/400/401/403/404/408/416/444(关连接)/500,默认 401 |
| 会话时长默认 | `sessionMaxAge` | 即时 | 分钟,默认 10080(7 天) |
| IP 限制白名单 | `ipLimitAllowlist` | 即时 | 数组,CIDR/单 IP,白名单内 IP 不受登录失败计数/封禁影响 |
| 镜像加速 | (独立接口) | 即时 | `/api/registry/mirrors`,写入 daemon.json registry-mirrors,保存后自动重启 Docker 服务 |

### 证书 tab

| 配置项 | JSON 字段 | 生效 | 说明 |
|--------|----------|------|------|
| 面板证书公钥文件路径 | `webCertFile` | 重启 | 容器内路径,如 /data/cert/fullchain.cer |
| 面板证书密钥文件路径 | `webKeyFile` | 重启 | 容器内路径,如 /data/cert/dk.kejizero.xyz.key |

> HTTPS 行为:证书路径有效则自动以 HTTPS 监听;`webForceSSL=true`(后端字段,UI 已隐藏)时证书无效将拒绝启动,false 时降级 HTTP 防失联。

### 日期和时间 tab
- 时区 `timeZone`(默认 Asia/Shanghai)、NTP 服务器 `ntpServer`(默认 pool.ntp.org)、日期类型 `datePickerType`。

### 安全 tab(管理员凭证 + 双重认证)
- 管理员凭证:**四字段**(原用户名/原密码/新用户名/新密码,原用户名预填当前账号,新字段留空 = 不修改)。
- 双重认证(TOTP):启用需当前密码,扫码/手动密钥,6 位动态码;禁用需密码+动态码。
- 密码策略(在常规 → 无独立 UI,由后端字段控制):`expirationDays`(密码过期天数,0=不过期,登录时强制改密)、`complexityVerification`(复杂度:8-64 位含大小写字母和数字,改密接口校验)。

### Telegram 机器人(仿 3x-ui)
- 子 tab:面板设置 | 通知。字段:`tgEnable/tgBotToken/tgAdminChatId/tgRunTime/tgBotBackup/tgLang/tgBotAPIServer/tgNotifyEvents`。
- 通知事件:`tgNotifyEvents` 数组(license/system/container/image/network/volume/groupAccount/groupDocker 分组)。

### 邮件设置
- 字段:`emailEnable/smtpHost/smtpPort/smtpUser/smtpPass/smtpFrom/smtpFromName/smtpTo/smtpEncryption/emailNotifyEvents`。加密方式 none/ssl/tls。

### 许可证
- 离线 Pro 许可证:文件上传激活(设备绑定)/反激活;免费版限制创建容器与 Compose 部署。
- API:`/api/license/status`、`/api/license/activate`(文件)、`/api/license/deactivate`。

### 关于
- 版本信息、项目地址、使用手册链接。

---

## 第四部分:安全特性

1. **webDomain Host 白名单**(Router 最外层中间件):设置后仅绑定域名可访问,其他 Host 404。用于"设置域名访问就关闭 IP 访问"。localhost/127.0.0.1/::1 永远放行(本地运维与 install.sh API 调用)。
2. **安全入口 webBasePath**:非入口路径 302 → 入口路径(保留路径,API/页面均带前缀可达);/assets、/logo.svg、/favicon.ico 直接服务。前端 router base 由 `GET /api/system/public-config` 动态获取(`main.js` 异步挂载),api.js 统一 `entrancePath()` 加前缀。
3. **未认证设置 noAuthSetting**:AuthMiddleware 401 时按配置返回指定状态码;444 额外加 `Connection: close`;200 返回帮助页 HTML(伪装正常服务)。
4. **登录失败事件**:登录失败写入事件记录(面板设置 → 日志)并触发 TG/邮件通知(`EvLoginFail`),用于排查暴力破解;无封禁机制。
5. **密码策略**:`expirationDays` 过期强制改密;`complexityVerification` 复杂度校验(错误 key:`user.pwdComplexity`)。
6. **JWT 认证**:Authorization: Bearer 头或 `?token=` query(WebSocket 用);401 自动跳登录页。

---

## 第五部分:安装脚本与域名 HTTPS

`install.sh`(仓库根目录)特性:
- **双模式**:compose 安装(推荐,含 docker-compose.yml 编排)与二进制安装。
- **重复执行检测**:已安装时显示版本/服务状态;`--force` 强制重装。
- **网络自动分流**:国内走加速站 + registry 加速,海外直连。
- **域名 HTTPS 绑定**:`bash install.sh ssl`(或安装时选域名绑定)→ acme.sh standalone 模式申请证书(需 80 端口空闲)→ 证书写入宿主 `/opt/docker-manager/cert/`(fullchain.cer + <域名>.key)→ 通过 API `PUT /api/system/settings` 写入 webDomain/webCertFile/webKeyFile 三字段 → compose 端口映射改为 443:8080(宿主 443 → 容器 8080)。
- **DNS 校验前置**(`check_domain_dns`):申请证书前强制校验域名 A 记录 == 本机公网 IP,不一致直接终止(防止 28.0.1.x 类事故)。
- **公网 IP 提取加固**(`is_public_ip`):排除内网/保留段(10.x/192.168.x/172.16-31.x/127.x/169.254.x/28.x.x.x DoD 保留段等);`DM_PUBLIC_IP=<ip>` 手动指定。
- 证书路径约定:宿主 `/opt/docker-manager/cert/` ↔ 容器内 `/data/cert/`。

---

## 第六部分:诊断命令与故障排查

> 交互模式:AI 给出命令 → 用户在 VPS 执行 → 粘贴输出 → AI 分析。🟢 安全查询 / 🟡 有副作用 / 🔴 高风险。

### 6.1 可达性诊断(🟢)
```bash
# DNS 解析是否指向公网 IP(154.201.92.245 示例;28.0.1.x = 错误)
dig +short <域名> @8.8.8.8
# VPS 端口是否通(本地)
curl -sI http://127.0.0.1:8080/ | head -3
curl -sI https://<域名>/ | head -3
# 从外网测(在本地电脑)
curl -sI https://<域名>/
```

### 6.2 容器状态(🟢)
```bash
docker compose ps                      # compose 安装
docker compose logs --tail 50          # 面板日志
docker ps -a | grep docker-manager     # 容器是否反复重启(restart 循环)
```

### 6.3 面板配置检查(🟢)
```bash
# 未认证公开接口
curl -s http://127.0.0.1:8080/api/system/public-config   # 返回 basePath(安全入口)
curl -s http://127.0.0.1:8080/api/health
# 容器内证书文件是否存在(证书路径写错 = https 起不来)
docker exec <容器名> ls -la /data/cert/
```

### 6.4 常见故障速查表

| 症状 | 原因 | 排查/修复 |
|------|------|----------|
| 域名打不开,IP 也打不开 | DNS A 记录错误(曾填 28.0.1.x 保留段)或防火墙未放行 | `dig` 确认解析到公网 IP;放行 80/443/8080 |
| https 打不开,报 wrong version number / SEC_E_INVALID_TOKEN | 证书路径没写进配置(旧 install.sh sed 静默失败)或容器内路径不存在 | `docker exec <c> ls /data/cert/`;重新 `bash install.sh ssl`;或 API PUT webCertFile/webKeyFile |
| 改了端口/安全入口不生效 | webPort/webBasePath 需重启面板 | 面板设置 → 保存 → 重启面板;或 `docker compose restart` |
| 设置了 webDomain 后 IP 访问 404 | Host 白名单生效(设计如此) | 用绑定域名访问;本机 localhost 不受限 |
| 设置了安全入口后旧链接打不开 | 302 到新入口,正常 | 用 /<入口>/ 访问;忘入口则重启并从 / 访问会 302 |
| 登录接口返回 401 | 用户名/密码错误(事件记录 login_fail) | 核对账号密码;登录失败会触发 TG/邮件通知 |
| 登录提示密码过期 | expirationDays 到期 | 设置 → 安全 → 修改密码 |
| 面板反复重启(restart loop) | webForceSSL=true 且证书无效 → 拒绝启动 | 修复证书路径或改回 false(证书 tab) |
| 镜像加速不生效 | daemon.json 写入后 Docker 重启失败 | 保存时提示;手动 `systemctl restart docker`(宿主机) |

### 6.5 运维提示
- 面板日志:compose 安装 `docker compose logs -f`;二进制安装 systemd journal `journalctl -u docker-manager -f`。
- SQLite 备份:`data/` 目录整体拷贝即可。
- 更新:`bash install.sh update`(compose 拉取最新镜像重建)。

---

## 第七部分:API 速查(常用)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/login | 登录(admin/密码),返回 JWT |
| POST | /api/login/totp | 2FA 验证 |
| GET | /api/system/public-config | 登录前:basePath(安全入口) |
| GET | /api/system/settings | 面板设置(secret 脱敏) |
| PUT | /api/system/settings | 补丁合并保存设置 |
| GET | /api/health | 健康检查 |
| GET/POST/DELETE | /api/containers... | 容器 CRUD/启停/终端 |
| GET/POST | /api/images... | 镜像列表/拉取/删除 |
| GET/POST | /api/networks... | 网络 |
| GET/POST | /api/volumes... | 卷 |
| GET/POST | /api/compose... | Compose 栈 |
| GET | /api/system/host | 宿主机信息 |
| GET | /api/system/registry-mirrors | 镜像加速配置 |
| PUT | /api/system/registry-mirrors | 保存镜像加速 |
| GET | /api/license/status | 许可证状态 |
| GET | /api/apps | 应用商店列表 + 分类(1Panel tags) |
| POST | /api/apps/sync | 同步应用商店数据(从 GitHub 仓库,~240MB) |
| GET | /api/apps/:key | 应用详情(版本列表 + 参数表单) |
| POST | /api/apps/:key/preview | 渲染 compose 预览 |
| POST | /api/apps/:key/install | 安装(参数 + 可选 yaml 覆盖) |
| POST | /api/apps/:key/upgrade | 升级(重渲染最新版本 + 重建) |
| POST | /api/apps/:key/uninstall | 卸载 |
| GET | /api/apps/icon/:key | 应用图标(公开接口,<img> 直接引用) |

> WebSocket(容器日志/终端/事件流):`ws(s)://<host>/api/<path>?token=<JWT>`;设置了安全入口时路径需带入口前缀(前端 api.js `entrancePath()` 自动处理)。

---

## 附:开发注意(写给维护者)

- 前端构建:`cd web && npm run build` → `go build` 内嵌 dist;修改前端必须重新 `go build` 才生效。
- 新增设置字段:改 `internal/settings/settings.go`(JSON tag)+ `Settings.Normalize()` 默认值 + 前端 SettingsView form/loadPanelSettings/savePanel + 14 个语言包(en/zh-CN/zh-TW/ja/ru/vi/es/id/uk/tr/pt-BR/ar/fa/ko)。
- 旧库兼容:SQLite 新增字段用 `ALTER TABLE ... ADD COLUMN` 迁移(db.go)。
- 出站请求(通知/公网 IP 查询)走 `netutil.HTTPClient(s, timeout)`(支持面板代理服务器设置)。
- 部署镜像:Github Actions(release.yml)构建多平台;Dockerfile 三阶段(node → golang CGO_ENABLED=0 → alpine + docker-compose 二进制)。
