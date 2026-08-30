[English](/README.md) | [فارسی](/README.fa_IR.md) | [العربية](/README.ar_EG.md) | [中文](/README.zh_CN.md) | [Español](/README.es_ES.md) | [Русский](/README.ru_RU.md) | [Türkçe](/README.tr_TR.md)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./media/docker-manager-dark.svg">
    <img alt="Docker Manager Go" src="./media/docker-manager-light.svg" width="850">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/DockOrae/DockOrae/releases"><img src="https://img.shields.io/github/v/release/DockOrae/DockOrae" alt="Release"></a>
  <a href="https://github.com/DockOrae/DockOrae/actions"><img src="https://img.shields.io/github/actions/workflow/status/DockOrae/DockOrae/release.yml.svg" alt="Build"></a>
  <a href="https://github.com/DockOrae/DockOrae/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/DockOrae/DockOrae.svg" alt="Go Version"></a>
  <a href="https://github.com/DockOrae/DockOrae/releases/latest"><img src="https://img.shields.io/github/downloads/DockOrae/DockOrae/total.svg" alt="Downloads"></a>
  <a href="https://hub.docker.com/r/zhaoweiwen123/dockorae"><img src="https://img.shields.io/docker/pulls/zhaoweiwen123/dockorae.svg" alt="Docker Pulls"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.en.html"><img src="https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true" alt="License"></a>
</p>

**Docker Manager Go** یک پنل مدیریت Docker مدرن و زیبا است که با زبان **Go** ([gin](https://github.com/gin-gonic/gin) + [Moby Docker SDK](https://github.com/moby/moby) رسمی) نوشته شده و فرانت‌اند آن با **Vue 3** ساخته شده است. طراحی رابط کاربری آن از طراحی تعاملی 1Panel الهام گرفته شده و دارای تم تیره/روشن با رنگ برند صورتی است؛ صفحه وضعیت سیستم نیز بر اساس 3x-ui طراحی شده است.

> [!IMPORTANT]
> این پروژه فقط برای استفاده شخصی طراحی شده است. لطفاً از آن برای اهداف غیرقانونی یا در محیط تولید بدون مجوز مناسب استفاده نکنید.

## امکانات

- **مدیریت کانتینرها** — ایجاد / شروع / توقف / راه‌اندازی مجدد / مکث / حذف / بازرسی / اتصال، همراه با **ترمینال کانتینر** داخلی (WebSocket).
- **مدیریت تصاویر** — دریافت (pull) با نمایش پیشرفت هم‌زمان، حذف و پاک‌سازی تصاویر بلااستفاده.
- **مدیریت شبکه** — ایجاد / حذف / بازرسی (پیکربندی زیرشبکه و دروازه).
- **مدیریت حجم‌ها** — ایجاد (محلی / NFS) / حذف / بازرسی.
- **فروشگاه برنامه** — بیش از ۲۶۰ برنامه با یک کلیک (مخزن سازگار با 1Panel: آیکون‌ها / فرم‌های پارامتر / چندنسخه)؛ همگام‌سازی خودکار در اولین راه‌اندازی (بدون اقدام دستی)، نصب / ارتقاء با یک کلیک با نشان به‌روزرسانی.
- **مدیریت استک‌های Compose** — ویرایشگر YAML، استقرار با یک کلیک (خروجی جریانی)، شروع/توقف و حذف.
- **پایش بلادرنگ** — صفحه وضعیت به سبک 3x-ui: کارت‌های CPU / حافظه / swap / ذخیره‌سازی با نمودارهای خطی، نمودارهای پهنای باند شبکه و ورودی/خروجی دیسک، تعداد کانتینرها/تصاویر/حجم‌ها، آمار فرایندهای پنل و IP عمومی با قابلیت نمایش/عدم نمایش.
- **لایسنس** — لایسنس آفلاین Pro (فعال‌سازی با آپلود فایل / اتصال دستگاه / لغو اتصال)؛ نسخه رایگان ایجاد کانتینر و استقرار Compose را محدود می‌کند.
- **آینه رجیستری** — پیکربندی registry-mirrors در `daemon.json` مستقیماً از داخل پنل.
- **چندزبانه** — ۱۴ زبان رابط کاربری با تم‌های تیره و روشن.
- **امنیت** — احراز هویت دو مرحله‌ای TOTP، نشست‌های JWT، آپلود آواتار.
- **جریان رویدادها** — رویدادهای بلادرنگ Docker که به داشبورد ارسال می‌شوند.

- **Panel settings (1Panel-inspired)** — security entrance path (panel accessible only via `/entrance`), unauthenticated response codes (200 help page / 400 / 401 / 403 / 404 / 408 / 416 / 444 / 500), panel domain whitelist (IP access disabled once a domain is bound), panel SSL certificate paths, password expiry & complexity policies, proxy server for outbound requests.
- **Toolbox** — device info, Docker disk cleanup (stopped containers / unused images & volumes / build cache), Fail2ban login protection with auto-ban, ban list and unban.

## 🤖 Agent Skill

This repository ships a built-in [GitHub Agent Skill](.github/skills/docker-manager-user-guide/SKILL.md) — `docker-manager-user-guide` — a knowledge base for AI assistants (Copilot / Claude / ChatGPT / etc.) covering panel configuration, deployment, and troubleshooting. Just ask your AI: How to configure the Docker Manager panel security entrance / domain binding / toolbox?

## شروع سریع

### نصب با یک دستور (پیشنهادی)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/DockOrae/DockOrae/master/install.sh)
```

نصب‌کننده:

- شبکه شما (داخلی/خارجی) را تشخیص می‌دهد و به‌طور خودکار از منابع شتاب‌دهنده استفاده می‌کند.
- اگر Docker نصب نباشد، آن را **به‌طور خودکار نصب می‌کند** (Debian / Ubuntu amd64 / arm64، آخرین نسخه پایدار).
- به شما امکان انتخاب روش نصب را می‌دهد:
  1. **Docker Compose** (پیشنهادی) — مبتنی بر تصویر، به‌روزرسانی آسان.
  2. **باینری محلی** (systemd) — بدون نیاز به Docker؛ معماری به‌طور خودکار تشخیص داده می‌شود (amd64 / arm64 / armv5-7 / 386 / s390x).
- به‌صورت اختیاری یک **دامنه با HTTPS** را متصل می‌کند (گواهی Let's Encrypt از طریق acme.sh).

دستورات پرکاربرد:

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

### Docker Compose (دستی)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  zhaoweiwen123/dockorae:latest
```

یا از `docker-compose.yml` همراه استفاده کنید. برای میزبان‌های Docker از راه دور، `DOCKER_HOST=tcp://<host>:2375` را تنظیم کنید.

### باینری (دستی)

فایل `docker-manager-go-linux-<arch>.tar.gz` را از [صفحه Releases](https://github.com/DockOrae/DockOrae/releases/latest) دانلود کنید، آن را استخراج کنید و اجرا کنید:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## اتصال دامنه (SSL)

پنل از HTTPS از طریق مسیرهای گواهی پیکربندی‌شده در **تنظیمات → عمومی → گواهی** پشتیبانی می‌کند. از منوی `ssl` نصب‌کننده برای صدور خودکار گواهی Let's Encrypt با acme.sh استفاده کنید (اعتبارسنجی standalone HTTP-01 — مطمئن شوید که دامنه به این دستگاه اشاره می‌کند و پورت ۸۰ آزاد است):

```bash
sudo bash install.sh ssl
```

مسیرهای گواهی به‌طور خودکار در تنظیمات پنل ذخیره می‌شوند و HTTPS پس از راه‌اندازی مجدد اعمال می‌شود.

## متغیرهای محیطی

| متغیر | پیش‌فرض | توضیحات |
|---|---|---|
| `DATA_DIR` | `./data` | دایرکتوری داده‌ها (پایگاه داده SQLite، تنظیمات، کاربران) |
| `PORT` | `8080` | پورت گوش دادن پنل |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | آدرس دیمن Docker |
| `TZ` | — | منطقه زمانی کانتینر |

## پلتفرم‌های پشتیبانی‌شده

- **باینری‌ها** (Linux): amd64, arm64, armv5, armv6, armv7, 386, s390x
- **تصاویر Docker**: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/s390x
- **محیط اجرای پنل**: Linux (تولید)، Windows (توسعه)

## زبان‌های پشتیبانی‌شده

English, 简体中文, 繁體中文, 日本語, 한국어, Русский, Türkçe, Español, Português (Brasil), Tiếng Việt, Indonesia, Українська, العربية, فارسی — ۱۴ زبان، با تشخیص خودکار و قابلیت جابه‌جایی با یک کلیک.

## لایسنس

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
