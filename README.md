# folder-together 📂⚡

[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)](https://golang.org)
[![Release](https://img.shields.io/github/v/release/hakankayaci-smilegames/folder-together?style=flat&color=38bdf8)](https://github.com/hakankayaci-smilegames/folder-together/releases)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows-blue)](https://github.com/hakankayaci-smilegames/folder-together)

> **Kubuntu (Linux) ve Windows 10 arasında sunucu kiralamadan (0 TL)**, hem yerel ağda (Gigabit LAN) hem de farklı internet ağlarındayken (Cloudflare Quick Tunnel) çalışan hızlı, güvenli, çift dilli dosya ve klasör paylaşım aracı.
>
> *Zero-server, cross-platform (Linux & Windows) P2P/Tunnel file sharing and folder hosting tool with native Desktop GUI and Web interface.*

---

## 🌟 Temel Özellikler / Key Features

- 🚀 **0 TL Maliyet / Sunucusuz (Serverless):** Farklı ağlardayken Cloudflare Quick Tunnel (`trycloudflare.com`) ile anında şifreli HTTPS bağlantısı açar. Port açma veya sunucu masrafı yoktur.
- ⚡ **Yerel Ağda Maksimum Hız (Gigabit LAN):** Aynı ağdaysanız doğrudan yerel IP üzerinden donanım hızında transfer sağlar.
- 🖥️ **Masaüstü Host Yönetici GUI (`/host`):**
  - Çift tıkla açılan modern yerel kontrol paneli.
  - **İşletim Sistemi Yerel Klasör Seçici:** Kubuntu'da KDE `kdialog` / `zenity`, Windows 10'da Windows `FolderBrowserDialog` penceresi açılır.
  - Canlı bağlantı kartları, tek tıkla kopyalama, QR kod gösterimi ve canlı aktivite günlüğü.
- 🌐 **Gelişmiş Alıcı Web Arayüzü (`/`):**
  - **Liste ve Izgara (Grid) Görünümü:** Dosyaları tablo veya kart formatında inceleme.
  - **Anlık Dosya Araması:** Dosya isimlerine göre harf harf filtreleme.
  - **Tarayıcı İçi Önizleme (Preview Modal):** Resimleri, kodları/metinleri, ses ve videoları indirmeden inceleme.
  - **Dinamik ZIP Akışı:** Klasörleri diske zip yazmadan HTTP stream üzerinden tek tıkla indirme.
  - **Çoklu Sürükle-Bırak Yükleme:** İlerleme yüzdesiyle dosya yükleme.
- 🔒 **Güvenlik & PIN:** Tek kullanımlık 4 haneli PIN koruması.
- 🌍 **Çift Dilli (i18n):** Türkçe (TR) ve İngilizce (EN) tam dil desteği.
- 📦 **Tek Binary (Single Static Binary):** Node.js, Electron, Python veya harici DLL gerektirmez.

---

## 📥 İndir / Download

En son derlenmiş sürümleri doğrudan [Releases](https://github.com/hakankayaci-smilegames/folder-together/releases) sayfasından indirebilirsiniz:
- **Linux (Kubuntu / Debian / Ubuntu / Arch):** `folder-together-linux-amd64`
- **Windows 10 / 11:** `folder-together-windows-amd64.exe`

---

## 🚀 Hızlı Başlangıç / Quick Start

### 1. Linux (Kubuntu)

```bash
# İndirip çalıştırma izni verin:
chmod +x folder-together-linux-amd64

# Masaüstü Arayüzüyle Başlatma (Varsayılan):
./folder-together-linux-amd64

# PIN Korumalı Başlatma:
./folder-together-linux-amd64 -pin auto

# Belirli bir klasörü paylaşma:
./folder-together-linux-amd64 -dir /yol/klasor -upload
```

### 2. Windows 10 / 11

`folder-together-windows-amd64.exe` dosyasına çift tıklamanız yeterlidir.
- Masaüstü paneli varsayılan tarayıcınızda açılır.
- **"Gözat..."** butonuna tıkladığınızda Windows'un yerel klasör seçme penceresi açılır.
- Linki kopyalayıp karşı tarafa gönderebilirsiniz.

---

## 🛠️ Kaynak Koddan Derleme / Build from Source

Go 1.23+ yüklü bir sistemde:

```bash
git clone https://github.com/hakankayaci-smilegames/folder-together.git
cd folder-together

# Hem Linux hem Windows (.exe) ikili dosyalarını derler:
make build

# Veya ayrı ayrı:
make build-linux
make build-windows
```

Çıktılar `bin/` klasöründe oluşacaktır.

---

## ⚙️ Komut Satırı Seçenekleri / CLI Flags

| Bayrak / Flag | Açıklama / Description | Varsayılan / Default |
| :--- | :--- | :--- |
| `-dir <path>` | Paylaşılacak klasör yolu / Path to share | `.` (Bulunulan dizin) |
| `-file <path>` | Tek dosya paylaşım modu / Single file mode | `""` |
| `-port <num>` | Dinlenecek port / Port number | `8080` (veya otomatik) |
| `-pin <code/auto>` | Erişim PIN kodu / Access PIN | `""` |
| `-upload` | Karşı tarafın dosya yüklemesine izin ver / Allow uploads | `false` |
| `-no-tunnel` | Yalnızca LAN modunda çalıştır / Disable internet tunnel | `false` |
| `-no-browser` | Tarayıcıyı otomatik açma / Don't auto-open browser | `false` |
| `-lang <tr/en>` | Arayüz dili / Interface language | Otomatik / Auto |

---

## 📐 Mimari Dokümantasyon

Detaylı teknik mimari akış diyagramları ve modül açıklamaları için [PROJECT_ARCHITECTURE.md](PROJECT_ARCHITECTURE.md) dosyasını inceleyebilirsiniz.

---

## 📄 Lisans / License

MIT License. Özgürce kullanabilir, geliştirebilir ve dağıtabilirsiniz.
