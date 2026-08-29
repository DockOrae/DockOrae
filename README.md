[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/docker-manager-dark.svg">
    <img alt="Docker Manager Go" src="./media/docker-manager-light.svg" width="850">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/DockerManger/Docker_Manager_Go/releases"><img src="https://img.shields.io/github/v/release/DockerManger/Docker_Manager_Go" alt="Release"></a>
  <a href="https://github.com/DockerManger/Docker_Manager_Go/actions"><img src="https://img.shields.io/github/actions/workflow/status/DockerManger/Docker_Manager_Go/release.yml.svg" alt="Build"></a>
  <a href="https://github.com/DockerManger/Docker_Manager_Go/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/DockerManger/Docker_Manager_Go.svg" alt="Go Version"></a>
  <a href="https://github.com/DockerManger/Docker_Manager_Go/releases/latest"><img src="https://img.shields.io/github/downloads/DockerManger/Docker_Manager_Go/total.svg" alt="Downloads"></a>
  <a href="https://hub.docker.com/r/zhaoweiwen123/docker-manager-go"><img src="https://img.shields.io/docker/pulls/zhaoweiwen123/docker-manager-go.svg" alt="Docker Pulls"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
</p>

**Docker Manager Go** is a modern, beautiful Docker management panel written in **Go** ([gin](https://github.com/gin-gonic/gin) + official [Moby Docker SDK](https://github.com/moby/moby)) with a **Vue 3** frontend. The UI is inspired by 1Panel's interaction design, featuring dark/light themes with a pink brand color, and the system status page is modeled after 3x-ui.

> [!IMPORTANT]
> This project is intended for personal use only. Please do not use it for illegal purposes or in a production environment without proper authorization.

## Features

- **Container management** — create / start / stop / restart / pause / delete / inspect / attach, with a built-in **container terminal** (WebSocket).
- **Image management** — pull with real-time progress, delete, and prune unused images.
- **Network management** — create / delete / inspect (subnet & gateway configuration).
- **Volume management** — create (**local / NFS**) / delete / inspect.
- **App Store** — 260+ one-click apps from a 1Panel-compatible repository (icons, parameter forms, multi-version); auto-synced on first start (no manual step), one-click install / upgrade with an "updatable" badge.
- **Compose stack management** — YAML editor, one-click deploy (streaming output), start/stop, and teardown.
- **Real-time monitoring** — 3x-ui style status page: CPU / memory / swap / storage cards with sparklines, network throughput & disk I/O curves, container/image/volume counts, panel process stats, and public IP with visibility toggle.
- **License** — online licensing powered by Docker_Manager_License (Ed25519 signed keys, device binding, 24h periodic verification, 7-day grace period, instant revocation); offline activation retained for existing users
- **Registry mirror** — configure `daemon.json` registry-mirrors right from the panel.
- **Multi-language** — 14 UI languages with dark and light themes.
- **Security** — TOTP two-factor authentication, JWT sessions, avatar upload.
- **Panel settings (1Panel-inspired)** — security entrance path (panel accessible only via `/entrance`), unauthenticated response codes (200 help page / 400 / 401 / 403 / 404 / 408 / 416 / 444 / 500), panel domain whitelist (IP access disabled once a domain is bound), panel SSL certificate paths, password expiry & complexity policies, proxy server for outbound requests.
- **Toolbox** — device info, Docker disk cleanup (stopped containers / unused images & volumes / build cache), Fail2ban login protection with auto-ban, ban list and unban.
- **Event stream** — real-time Docker events pushed to the dashboard.
- **Online update** — auto-checks GitHub Releases (badge on the version icon in the footer when a new version is available) and updates with one click for both deployment modes: compose (independent helper container re-pulls & recreates the panel) or binary (atomic self-replacement + systemd restart).

## 🤖 Agent Skill

This repository ships a built-in [GitHub Agent Skill](.github/skills/docker-manager-user-guide/SKILL.md) — `docker-manager-user-guide` — a knowledge base for AI assistants (Copilot / Claude / ChatGPT / etc.) covering panel configuration, deployment, and troubleshooting. Just ask your AI: *"How to configure the Docker Manager panel security entrance / domain binding / toolbox?"*

## Quick Start

### One-line install (recommended)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/DockerManger/Docker_Manager_Go/master/install.sh)
```

The installer:

- Detects your network (domestic/overseas) and uses accelerated sources automatically.
- **Auto-installs Docker** if it is missing (Debian / Ubuntu amd64 / arm64, latest stable).
- Lets you choose the install method:
  1. **Docker Compose** (recommended) — image based, easy updates.
  2. **Local binary** (systemd) — no Docker required; architecture auto-detected (amd64 / arm64 / armv5-7 / 386 / s390x).
- Optionally binds a **domain with HTTPS** (Let's Encrypt certificate via acme.sh).

Common commands:

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

### Docker Compose (manual)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  zhaoweiwen123/docker-manager-go:latest
```

Or use the bundled `docker-compose.yml`. For remote Docker hosts, set `DOCKER_HOST=tcp://<host>:2375`.

### Binary (manual)

Download `docker-manager-go-linux-<arch>.tar.gz` from the [Releases page](https://github.com/DockerManger/Docker_Manager_Go/releases/latest), extract it, and run:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## Domain Binding (SSL)

The panel supports HTTPS via certificate paths configured in **Settings → General → Certificate**. Use the installer's `ssl` menu to issue a Let's Encrypt certificate automatically with acme.sh (HTTP-01 standalone validation — make sure the domain resolves to this machine and port 80 is free):

```bash
sudo bash install.sh ssl
```

The certificate paths are written into the panel settings automatically and HTTPS takes effect after restart.

## Online Licensing (Pro)

Pro features (Compose stacks, container creation, App Store installs) are gated by an online license:
the panel verifies a signed Ed25519 license key against a License Server, with device binding,
24h periodic verification, a 7-day grace period, and instant revocation.

Security model (V3):
- **License Key** is used only for first activation / re-activation (local Ed25519 verify)
- Runtime verification uses an **Activation Token** (stored locally in `license.json`, mode 0600;
  the server keeps only a SHA-256 hash — never the plaintext)
- Every verify/deactivate carries `timestamp + nonce` (replay protection)
- The server returns `server_time` on every response; the panel maintains a `clock_offset`
  (trusted time) and detects local **clock rollback** (>5min → Pro disabled)
- Server-side version control: `minimum_client_version` (upgrade prompt) and
  `blocked_versions` (emergency block → Pro disabled)

### 1. Deploy the License Server

Deploy [Docker_Manager_License](https://github.com/DockerManger/Docker_Manager_License) on any
server — single container, port 80, zero config. Step-by-step guides for both **direct IP** and
**domain + Cloudflare HTTPS** setups: **[docs/DEPLOY.md](https://github.com/DockerManger/Docker_Manager_License/blob/master/docs/DEPLOY.md)**.

> ⚠️ Use the private key paired with this panel's built-in public key (`private/license.key` in the
> License repo) — otherwise the panel's signature verification will fail. See deploy guide step 3.

### 2. Point the panel at your License Server

By default the panel uses the official server `https://manager.kejizero.xyz/license-api`
(nothing to configure). For a self-hosted License Server, set the environment variable:

| Variable | Default | Description |
|---|---|---|
| `DM_LICENSE_SERVER_URL` | `https://manager.kejizero.xyz/license-api` | License Server base URL, e.g. `http://<ip>/license-api` or `https://license.example.com/license-api`. Empty string = offline mode (legacy keys only). |

### 3. Activate

Panel → **Settings → Licensing** → **Add** → paste the License Key issued by the License admin
panel → **Activate**. The status badge shows the online verification state; click **Verify now**
to pick up a revocation instantly (otherwise within 24h).

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `DATA_DIR` | `./data` | Data directory (SQLite database, settings, users) |
| `PORT` | `8080` | Panel listen port |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | Docker daemon address |
| `TZ` | — | Time zone for the container |

## Supported Platforms

- **Binaries** (Linux): amd64, arm64, armv5, armv6, armv7, 386, s390x
- **Docker images**: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/s390x
- **Panel runtime**: Linux (production), Windows (development)

## Supported Languages

English, 简体中文, 繁體中文, 日本語, 한국어, Русский, Türkçe, Español, Português (Brasil), Tiếng Việt, Indonesia, Українська, العربية, فارسی — 14 languages, auto-detected with one-click switching.

## License

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
