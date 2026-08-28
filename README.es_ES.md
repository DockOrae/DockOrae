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

**Docker Manager Go** es un panel de gestión de Docker moderno y elegante escrito en **Go** ([gin](https://github.com/gin-gonic/gin) + el [Moby Docker SDK](https://github.com/moby/moby) oficial) con un frontend en **Vue 3**. La interfaz está inspirada en el diseño de interacción de 1Panel, con temas claro y oscuro y un color de marca rosa, y la página de estado del sistema está modelada según 3x-ui.

> [!IMPORTANT]
> Este proyecto está pensado únicamente para uso personal. No lo utilice con fines ilegales ni en un entorno de producción sin la debida autorización.

## Funciones

- **Gestión de contenedores** — crear / iniciar / detener / reiniciar / pausar / eliminar / inspeccionar / adjuntar, con una **terminal de contenedor** integrada (WebSocket).
- **Gestión de imágenes** — descargar con progreso en tiempo real, eliminar y limpiar imágenes no utilizadas.
- **Gestión de redes** — crear / eliminar / inspeccionar (configuración de subred y puerta de enlace).
- **Gestión de volúmenes** — crear (local / NFS) / eliminar / inspeccionar.
- **Tienda de aplicaciones** — más de 260 aplicaciones con un clic (repositorio compatible con 1Panel: iconos / formularios de parámetros / multiversión); sincronización desde GitHub, instalación / actualización con un clic e insignia de actualización disponible.
- **Gestión de stacks Compose** — editor YAML, despliegue con un clic (salida en streaming), iniciar/detener y desmontar.
- **Monitorización en tiempo real** — página de estado al estilo 3x-ui: tarjetas de CPU / memoria / swap / almacenamiento con minigráficos, curvas de rendimiento de red y E/S de disco, conteos de contenedores/imágenes/volúmenes, estadísticas de procesos del panel e IP pública con conmutador de visibilidad.
- **Licencia** — licencia Pro sin conexión (activación por carga de archivo / vinculación de dispositivo / desvinculación); el plan gratuito limita la creación de contenedores y el despliegue con Compose.
- **Espejo del registro** — configure los espejos de registro de `daemon.json` directamente desde el panel.
- **Multilingüe** — 14 idiomas de interfaz con temas claro y oscuro.
- **Seguridad** — autenticación de dos factores TOTP, sesiones JWT y carga de avatar.
- **Flujo de eventos** — eventos de Docker en tiempo real enviados al panel de control.

- **Panel settings (1Panel-inspired)** — security entrance path (panel accessible only via `/entrance`), unauthenticated response codes (200 help page / 400 / 401 / 403 / 404 / 408 / 416 / 444 / 500), panel domain whitelist (IP access disabled once a domain is bound), panel SSL certificate paths, password expiry & complexity policies, proxy server for outbound requests.
- **Toolbox** — device info, Docker disk cleanup (stopped containers / unused images & volumes / build cache), Fail2ban login protection with auto-ban, ban list and unban.

## 🤖 Agent Skill

This repository ships a built-in [GitHub Agent Skill](.github/skills/docker-manager-user-guide/SKILL.md) — `docker-manager-user-guide` — a knowledge base for AI assistants (Copilot / Claude / ChatGPT / etc.) covering panel configuration, deployment, and troubleshooting. Just ask your AI: How to configure the Docker Manager panel security entrance / domain binding / toolbox?

## Inicio rápido

### Instalación con un solo comando (recomendada)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/install.sh)
```

El instalador:

- Detecta su red (nacional/internacional) y utiliza fuentes aceleradas automáticamente.
- **Instala Docker automáticamente** si no está presente (Debian / Ubuntu amd64 / arm64, última versión estable).
- Le permite elegir el método de instalación:
  1. **Docker Compose** (recomendado) — basado en imágenes, actualizaciones sencillas.
  2. **Binario local** (systemd) — no requiere Docker; arquitectura detectada automáticamente (amd64 / arm64 / armv5-7 / 386 / s390x).
- Opcionalmente vincula un **dominio con HTTPS** (certificado de Let's Encrypt mediante acme.sh).

Comandos habituales:

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

O utilice el `docker-compose.yml` incluido. Para hosts Docker remotos, establezca `DOCKER_HOST=tcp://<host>:2375`.

### Binario (manual)

Descargue `docker-manager-go-linux-<arch>.tar.gz` desde la [página de Releases](https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest), extráigalo y ejecute:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## Vinculación de dominio (SSL)

El panel admite HTTPS mediante rutas de certificado configuradas en **Ajustes → General → Certificado**. Utilice el menú `ssl` del instalador para emitir automáticamente un certificado de Let's Encrypt con acme.sh (validación HTTP-01 independiente — asegúrese de que el dominio resuelva a esta máquina y de que el puerto 80 esté libre):

```bash
sudo bash install.sh ssl
```

Las rutas de los certificados se escriben automáticamente en los ajustes del panel y HTTPS surte efecto tras reiniciar.

## Variables de entorno

| Variable | Valor predeterminado | Descripción |
|---|---|---|
| `DATA_DIR` | `./data` | Directorio de datos (base de datos SQLite, ajustes, usuarios) |
| `PORT` | `8080` | Puerto de escucha del panel |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | Dirección del daemon de Docker |
| `TZ` | — | Zona horaria del contenedor |

## Plataformas compatibles

- **Binarios** (Linux): amd64, arm64, armv5, armv6, armv7, 386, s390x
- **Imágenes Docker**: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/s390x
- **Entorno de ejecución del panel**: Linux (producción), Windows (desarrollo)

## Idiomas compatibles

English, 简体中文, 繁體中文, 日本語, 한국어, Русский, Türkçe, Español, Português (Brasil), Tiếng Việt, Indonesia, Українська, العربية, فارسی — 14 idiomas, con detección automática y cambio con un solo clic.

## Licencia

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
