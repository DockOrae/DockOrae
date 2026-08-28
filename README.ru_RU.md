[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<p align="center">
  <img alt="Docker Manager Go" src="https://img.shields.io/badge/Docker%20Manager-Go-ec4899?logo=docker&logoColor=white&labelColor=0f172a">
</p>

<p align="center">
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/releases"><img src="https://img.shields.io/github/v/release/MinimaxFlora/Docker_Manager_Go" alt="Release"></a>
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/actions"><img src="https://img.shields.io/github/actions/workflow/status/MinimaxFlora/Docker_Manager_Go/release.yml.svg" alt="Build"></a>
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/MinimaxFlora/Docker_Manager_Go.svg" alt="Go Version"></a>
  <a href="https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest"><img src="https://img.shields.io/github/downloads/MinimaxFlora/Docker_Manager_Go/total.svg" alt="Downloads"></a>
  <a href="https://hub.docker.com/r/zhaoweiwen123/docker-manager-go"><img src="https://img.shields.io/docker/pulls/zhaoweiwen123/docker-manager-go.svg" alt="Docker Pulls"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
</p>

**Docker Manager Go** — это современная, красивая панель управления Docker, написанная на **Go** ([gin](https://github.com/gin-gonic/gin) + официальный [Moby Docker SDK](https://github.com/moby/moby)) с фронтендом на **Vue 3**. Интерфейс вдохновлён дизайном взаимодействия 1Panel: поддерживаются тёмная и светлая темы с розовым фирменным цветом, а страница состояния системы выполнена в стиле 3x-ui.

> [!IMPORTANT]
> Этот проект предназначен только для личного использования. Пожалуйста, не используйте его в незаконных целях или в производственной среде без соответствующего разрешения.

## Возможности

- **Управление контейнерами** — create / start / stop / restart / pause / delete / inspect / attach, со встроенным **веб-терминалом**.
- **Управление образами** — загрузка (pull) с прогрессом в реальном времени, удаление и очистка (prune) неиспользуемых образов.
- **Управление сетями** — создание / удаление / осмотр (настройка подсети и шлюза).
- **Управление томами** — создание / удаление / осмотр.
- **Управление стеками Compose** — редактор YAML, развёртывание в один клик (потоковый вывод), запуск/остановка и удаление стека.
- **Мониторинг в реальном времени** — страница состояния в стиле 3x-ui: карточки CPU / памяти / swap / хранилища со спарклайнами, графики пропускной способности сети и дискового ввода-вывода, счётчики контейнеров/образов/томов, статистика процессов панели и публичный IP с переключателем видимости.
- **Терминал** — терминал хоста (chroot `/host`), терминал контейнера и **управление SSH-хостами** (группы / подключение / авторизация по паролю и ключу), быстрые команды, настройки внешнего вида терминала.
- **Лицензия** — офлайн-лицензия Pro (активация загрузкой файла / привязка устройства / отвязка); бесплатный тариф ограничивает создание контейнеров и развёртывание Compose.
- **Зеркало реестра** — настройка зеркал реестра в `daemon.json` прямо из панели.
- **Многоязычность** — 14 языков интерфейса с тёмной и светлой темами.
- **Безопасность** — двухфакторная аутентификация TOTP, сессии JWT, загрузка аватара.
- **Поток событий** — события Docker в реальном времени, передаваемые на панель управления.

## Быстрый старт

### Установка одной командой (рекомендуется)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/install.sh)
```

Установщик:

- Определяет вашу сеть (внутреннюю/зарубежную) и автоматически использует ускоренные источники.
- **Автоматически устанавливает Docker**, если он отсутствует (Debian / Ubuntu amd64 / arm64, последняя стабильная версия).
- Позволяет выбрать способ установки:
  1. **Docker Compose** (рекомендуется) — на основе образа, простое обновление.
  2. **Локальный бинарный файл** (systemd) — Docker не требуется; архитектура определяется автоматически (amd64 / arm64 / armv5-7 / 386 / s390x).
- Опционально привязывает **домен с HTTPS** (сертификат Let's Encrypt через acme.sh).

Часто используемые команды:

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

### Docker Compose (вручную)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  zhaoweiwen123/docker-manager-go:latest
```

Или используйте встроенный файл `docker-compose.yml`. Для удалённых Docker-хостов задайте `DOCKER_HOST=tcp://<host>:2375`.

### Бинарный файл (вручную)

Скачайте `docker-manager-go-linux-<arch>.tar.gz` со [страницы релизов](https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest), распакуйте его и запустите:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## Привязка домена (SSL)

Панель поддерживает HTTPS через пути к сертификатам, настраиваемые в разделе **Настройки → Общие → Сертификат**. Используйте меню `ssl` установщика, чтобы автоматически выпустить сертификат Let's Encrypt с помощью acme.sh (автономная проверка HTTP-01 — убедитесь, что домен указывает на эту машину и порт 80 свободен):

```bash
sudo bash install.sh ssl
```

Пути к сертификатам автоматически записываются в настройки панели, и HTTPS вступает в силу после перезапуска.

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `DATA_DIR` | `./data` | Каталог данных (база данных SQLite, настройки, пользователи) |
| `PORT` | `8080` | Порт прослушивания панели |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | Адрес демона Docker |
| `TZ` | — | Часовой пояс контейнера |

## Поддерживаемые платформы

- **Бинарные файлы** (Linux): amd64, arm64, armv5, armv6, armv7, 386, s390x
- **Образы Docker**: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/s390x
- **Среда выполнения панели**: Linux (продакшн), Windows (разработка)

## Поддерживаемые языки

English, 简体中文, 繁體中文, 日本語, 한국어, Русский, Türkçe, Español, Português (Brasil), Tiếng Việt, Indonesia, Українська, العربية, فارسی — 14 языков с автоматическим определением и переключением в один клик.

## Лицензия

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
