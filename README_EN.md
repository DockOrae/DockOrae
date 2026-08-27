# Docker Manager (Go)

> **中文**: [README.md](README.md)

A beautiful Docker management panel written in **Go** (gin + official Docker SDK) with a **Vue 3** frontend. Dark theme with pink brand color, card-style dashboard. Features:

- 📦 **Containers**: create / start / stop / restart / pause / remove / inspect
- 🖼️ **Images**: pull (live progress) / remove / inspect / prune
- 🌐 **Networks**: create / remove / inspect (subnet & gateway support)
- 💾 **Volumes**: create / remove / inspect
- 🧩 **Compose stacks**: YAML editor / one-click deploy / start-stop / down / live logs
- 📊 **Live monitoring**: WebSocket real-time logs, CPU / memory / network charts
- 🖥️ **Web terminal**: open a shell inside the container right in your browser (TTY)
- 🔔 **Event stream**: real-time Docker events with auto-refreshing dashboard

## Quick start

```bash
curl -o docker-compose.yml https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/docker-compose.yml
docker compose up -d
```

Or create `docker-compose.yml` manually:

```yaml
services:
  docker-manager:
    image: zhaoweiwen123/docker-manager-go:latest
    container_name: docker-manager
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - docker-manager-data:/data
    environment:
      - TZ=Asia/Shanghai

volumes:
  docker-manager-data:
```

Open `http://<host-ip>:8080` — default credentials **admin / 123456** (please change them in Settings after first login).

> ⚠️ The panel talks to the Docker daemon through the mounted socket and therefore has root-equivalent Docker privileges. Do not expose it to the public internet.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | Panel listen port |
| `DATA_DIR` | `/data` | Data dir (users, secrets, compose files) |
| `COMPOSE_BIN` | `docker-compose` | Compose binary path (bundled in the image) |

## Tech stack

- **Backend**: Go · axum 0.8 · bollard (Docker Engine API) · argon2 (JWT sessions)
- **Frontend**: Vue 3 · Vite · Tailwind CSS 4 · xterm.js (web terminal)
- **Deploy**: multi-stage Docker build (amd64 / arm64) · GitHub Actions auto-push to Docker Hub

## Local development

```bash
# frontend (HMR, proxies to :8080)
cd web && npm install && npm run dev

# backend (Go toolchain required; no C compiler needed on Windows)
cargo run
# without a local Docker daemon, point DATA_DIR somewhere writable:
DATA_DIR=./data cargo run
```

## License

MIT
