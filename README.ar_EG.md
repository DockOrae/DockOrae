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

**Docker Manager Go** هي لوحة تحكم حديثة وجميلة لإدارة Docker مكتوبة بلغة **Go** ([gin](https://github.com/gin-gonic/gin) + [Moby Docker SDK](https://github.com/moby/moby) الرسمي) مع واجهة أمامية مبنية بـ **Vue 3**. واجهة المستخدم مستوحاة من تصميم تفاعل 1Panel، وتتميز بثيمات داكنة/فاتحة مع لون علامة تجارية وردي، وصفحة حالة النظام مستوحاة من 3x-ui.

> [!IMPORTANT]
> هذا المشروع مخصص للاستخدام الشخصي فقط. يُرجى عدم استخدامه لأغراض غير قانونية أو في بيئة إنتاج دون تصريح مناسب.

## الميزات

- **إدارة الحاويات** — إنشاء / تشغيل / إيقاف / إعادة تشغيل / إيقاف مؤقت / حذف / فحص / إرفاق، مع **طرفية حاوية** مدمجة (WebSocket).
- **إدارة الصور** — سحب الصور مع عرض التقدم لحظيًا، وحذف الصور غير المستخدمة وتنظيفها.
- **إدارة الشبكات** — إنشاء / حذف / فحص (تكوين الشبكة الفرعية والبوابة).
- **إدارة وحدات التخزين** — إنشاء (محلي / NFS) / حذف / فحص.
- **متجر التطبيقات** — أكثر من 260 تطبيقًا بنقرة واحدة (مستودع متوافق مع 1Panel: أيقونات / نماذج معلمات / إصدارات متعددة)؛ مزامنة من GitHub، تثبيت / ترقية بنقرة واحدة مع شارة التحديث المتاح.
- **إدارة مجموعات Compose** — محرر YAML، ونشر بنقرة واحدة (مع بث المخرجات)، وتشغيل/إيقاف، وإزالة.
- **مراقبة لحظية** — صفحة حالة بأسلوب 3x-ui: بطاقات CPU / الذاكرة / swap / التخزين مع رسوم بيانية مصغرة، ومنحنيات معدل نقل الشبكة و I/O القرص، وأعداد الحاويات/الصور/وحدات التخزين، وإحصائيات عمليات اللوحة، والعنوان IP العام مع خيار إظهار/إخفاء.
- **الترخيص** — ترخيص Pro دون اتصال (تفعيل برفع ملف / ربط الجهاز / إلغاء الربط)؛ الطبقة المجانية تحدّ من إنشاء الحاويات ونشر Compose.
- **مرآة السجل** — تكوين مرايا السجل في `daemon.json` مباشرة من اللوحة.
- **تعدد اللغات** — 14 لغة للواجهة مع ثيمات داكنة وفاتحة.
- **الأمان** — مصادقة ثنائية TOTP، وجلسات JWT، ورفع الصورة الرمزية.
- **بث الأحداث** — أحداث Docker لحظية تُدفع إلى لوحة المعلومات.

- **Panel settings (1Panel-inspired)** — security entrance path (panel accessible only via `/entrance`), unauthenticated response codes (200 help page / 400 / 401 / 403 / 404 / 408 / 416 / 444 / 500), panel domain whitelist (IP access disabled once a domain is bound), panel SSL certificate paths, password expiry & complexity policies, proxy server for outbound requests.
- **Toolbox** — device info, Docker disk cleanup (stopped containers / unused images & volumes / build cache), Fail2ban login protection with auto-ban, ban list and unban.

## 🤖 Agent Skill

This repository ships a built-in [GitHub Agent Skill](.github/skills/docker-manager-user-guide/SKILL.md) — `docker-manager-user-guide` — a knowledge base for AI assistants (Copilot / Claude / ChatGPT / etc.) covering panel configuration, deployment, and troubleshooting. Just ask your AI: How to configure the Docker Manager panel security entrance / domain binding / toolbox?

## بدء سريع

### تثبيت بسطر واحد (موصى به)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/install.sh)
```

المثبّت:

- يكتشف شبكتك (محلية/خارجية) ويستخدم مصادر متسارعة تلقائيًا.
- **يثبّت Docker تلقائيًا** إذا كان غير موجود (Debian / Ubuntu amd64 / arm64، أحدث إصدار مستقر).
- يتيح لك اختيار طريقة التثبيت:
  1. **Docker Compose** (موصى به) — يعتمد على الصور، وتحديثات سهلة.
  2. **ثنائي محلي** (systemd) — لا يتطلب Docker؛ يتم اكتشاف البنية تلقائيًا (amd64 / arm64 / armv5-7 / 386 / s390x).
- يربط اختياريًا **نطاقًا مع HTTPS** (شهادة Let's Encrypt عبر acme.sh).

الأوامر الشائعة:

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

### Docker Compose (يدوي)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  zhaoweiwen123/docker-manager-go:latest
```

أو استخدم ملف `docker-compose.yml` المرفق. لمضيفات Docker البعيدة، اضبط `DOCKER_HOST=tcp://<host>:2375`.

### ثنائي (يدوي)

نزّل `docker-manager-go-linux-<arch>.tar.gz` من [صفحة الإصدارات](https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest)، ثم فك ضغطه وشغّله:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## ربط النطاق (SSL)

تدعم اللوحة HTTPS عبر مسارات الشهادات المكوّنة في **الإعدادات ← عام ← الشهادة**. استخدم قائمة `ssl` في المثبّت لإصدار شهادة Let's Encrypt تلقائيًا باستخدام acme.sh (تحقق مستقل HTTP-01 — تأكد من أن النطاق يحل إلى هذا الجهاز وأن المنفذ 80 متاح):

```bash
sudo bash install.sh ssl
```

تُكتب مسارات الشهادات تلقائيًا في إعدادات اللوحة ويصبح HTTPS ساريًا بعد إعادة التشغيل.

## متغيرات البيئة

| المتغير | الافتراضي | الوصف |
|---|---|---|
| `DATA_DIR` | `./data` | دليل البيانات (قاعدة بيانات SQLite، الإعدادات، المستخدمون) |
| `PORT` | `8080` | منفذ استماع اللوحة |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | عنوان خادم Docker |
| `TZ` | — | المنطقة الزمنية للحاوية |

## المنصات المدعومة

- **الثنائيات** (Linux): amd64, arm64, armv5, armv6, armv7, 386, s390x
- **صور Docker**: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/s390x
- **بيئة تشغيل اللوحة**: Linux (إنتاج)، Windows (تطوير)

## اللغات المدعومة

English, 简体中文, 繁體中文, 日本語, 한국어, Русский, Türkçe, Español, Português (Brasil), Tiếng Việt, Indonesia, Українська, العربية, فارسی — 14 لغة، مع اكتشاف تلقائي وتبديل بنقرة واحدة.

## الترخيص

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
