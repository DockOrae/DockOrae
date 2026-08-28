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

**Docker Manager Go**, **Go** ([gin](https://github.com/gin-gonic/gin) + resmî [Moby Docker SDK](https://github.com/moby/moby)) ile yazılmış ve **Vue 3** ön yüze sahip modern, şık bir Docker yönetim panelidir. Arayüz, 1Panel'in etkileşim tasarımından esinlenmiştir; pembe marka rengiyle koyu/açık temalar sunar ve sistem durum sayfası 3x-ui'den modellenmiştir.

> [!IMPORTANT]
> Bu proje yalnızca kişisel kullanım içindir. Lütfen yasadışı amaçlarla veya uygun yetkilendirme olmadan üretim ortamında kullanmayın.

## Özellikler

- **Konteyner yönetimi** — oluşturma / başlatma / durdurma / yeniden başlatma / duraklatma / silme / inceleme / bağlanma, yerleşik **Web terminali** ile.
- **İmaj yönetimi** — gerçek zamanlı ilerlemeyle çekme (pull), silme ve kullanılmayan imajları temizleme.
- **Ağ yönetimi** — oluşturma / silme / inceleme (alt ağ ve ağ geçidi yapılandırması).
- **Birim (volume) yönetimi** — oluşturma / silme / inceleme.
- **Compose yığın yönetimi** — YAML düzenleyici, tek tıkla dağıtım (akış çıktısı), başlatma/durdurma ve kaldırma.
- **Gerçek zamanlı izleme** — 3x-ui tarzı durum sayfası: mini grafikli CPU / bellek / takas / depolama kartları, ağ bant genişliği ve disk G/Ç eğrileri, konteyner/imaj/birim sayıları, panel süreç istatistikleri ve görünürlük açma/kapama özellikli genel IP.
- **Terminal** — ana makine terminali (chroot `/host`), konteyner terminali ve **SSH ana makine yönetimi** (gruplar / bağlanma / parola ve anahtar kimlik doğrulaması), hızlı komutlar, terminal görünüm ayarları.
- **Lisans** — çevrimdışı Pro lisansı (dosya yükleme ile etkinleştirme / cihaz bağlama / bağlama kaldırma); ücretsiz sürüm konteyner oluşturmayı ve Compose dağıtımını sınırlar.
- **Kayıt defteri aynası (registry mirror)** — `daemon.json` kayıt defteri aynalarını doğrudan panelden yapılandırın.
- **Çok dilli** — koyu ve açık temalarla 14 arayüz dili.
- **Güvenlik** — TOTP iki faktörlü kimlik doğrulama, JWT oturumları, avatar yükleme.
- **Olay akışı** — gerçek zamanlı Docker olayları panoya iletilir.

## Hızlı Başlangıç

### Tek satırlık kurulum (önerilen)

```bash
bash <(curl -Ls https://raw.githubusercontent.com/MinimaxFlora/Docker_Manager_Go/master/install.sh)
```

Kurulum programı:

- Ağınızı (yurt içi/yurt dışı) algılar ve hızlandırılmış kaynakları otomatik kullanır.
- Eksikse **Docker'ı otomatik kurar** (Debian / Ubuntu amd64 / arm64, en son kararlı sürüm).
- Kurulum yöntemini seçmenize olanak tanır:
  1. **Docker Compose** (önerilen) — imaj tabanlı, kolay güncelleme.
  2. **Yerel ikili dosya (binary)** (systemd) — Docker gerektirmez; mimari otomatik algılanır (amd64 / arm64 / armv5-7 / 386 / s390x).
- İsteğe bağlı olarak **HTTPS ile bir alan adı** bağlar (acme.sh aracılığıyla Let's Encrypt sertifikası).

Yaygın komutlar:

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

### Docker Compose (elle)

```bash
docker run -d --name docker-manager-go \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v docker-manager-data:/data \
  zhaoweiwen123/docker-manager-go:latest
```

Ya da birlikte gelen `docker-compose.yml` dosyasını kullanın. Uzak Docker ana makineleri için `DOCKER_HOST=tcp://<host>:2375` değerini ayarlayın.

### İkili dosya (binary) (elle)

`docker-manager-go-linux-<arch>.tar.gz` dosyasını [Sürümler sayfası](https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest) bölümünden indirin, çıkarın ve çalıştırın:

```bash
tar xzf docker-manager-go-linux-amd64.tar.gz
sudo mv docker-manager-go/docker-manager-go /usr/local/bin/
DATA_DIR=/opt/docker-manager/data PORT=8080 docker-manager-go
```

## Alan Adı Bağlama (SSL)

Panel, **Ayarlar → Genel → Sertifika** bölümünde yapılandırılan sertifika yollarıyla HTTPS'i destekler. Let's Encrypt sertifikasını acme.sh ile otomatik olarak almak için kurulum programının `ssl` menüsünü kullanın (HTTP-01 bağımsız doğrulama — alan adının bu makineye çözümlendiğinden ve 80 numaralı bağlantı noktasının boş olduğundan emin olun):

```bash
sudo bash install.sh ssl
```

Sertifika yolları panel ayarlarına otomatik olarak yazılır ve HTTPS, yeniden başlatmanın ardından etkili olur.

## Ortam Değişkenleri

| Değişken | Varsayılan | Açıklama |
|---|---|---|
| `DATA_DIR` | `./data` | Veri dizini (SQLite veritabanı, ayarlar, kullanıcılar) |
| `PORT` | `8080` | Panel dinleme bağlantı noktası |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` (Linux) | Docker daemon adresi |
| `TZ` | — | Konteyner için saat dilimi |

## Desteklenen Platformlar

- **İkili dosyalar (binary)** (Linux): amd64, arm64, armv5, armv6, armv7, 386, s390x
- **Docker imajları**: linux/amd64, linux/arm64, linux/arm/v7, linux/arm/v6, linux/s390x
- **Panel çalışma zamanı**: Linux (üretim), Windows (geliştirme)

## Desteklenen Diller

İngilizce, 简体中文, 繁體中文, 日本語, 한국어, Русский, Türkçe, Español, Português (Brasil), Tiếng Việt, Indonesia, Українська, العربية, فارسی — 14 dil, otomatik algılama ve tek tıkla değiştirme.

## Lisans

[GPL V3](https://www.gnu.org/licenses/gpl-3.0.en.html)
