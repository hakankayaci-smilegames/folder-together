package i18n

import (
	"os"
	"strings"
)

type Language string

const (
	LangTR Language = "tr"
	LangEN Language = "en"
)

var currentLang = LangTR

var translations = map[Language]map[string]string{
	LangTR: {
		// App & General
		"app_title":            "folder-together",
		"app_desc":             "Kubuntu & Windows 10 Hızlı ve Güvenli Dosya Paylaşımı",
		"status_active":        "Paylaşım Aktif",
		"status_stopped":       "Paylaşım Durduruldu",
		"sharing_folder":       "Paylaşılan Klasör",
		"sharing_file":         "Paylaşılan Dosya",
		"local_url":            "Yerel Ağ (LAN) Adresi",
		"tunnel_url":           "İnternet / Tünel Adresi (Farklı Ağlar)",
		"pin_code":             "Erişim PIN Kodu",
		"pin_required":         "Bu paylaşıma erişmek için PIN kodu gereklidir",
		"pin_placeholder":      "PIN kodunu girin",
		"pin_submit":           "Giriş Yap",
		"pin_invalid":          "Hatalı PIN kodu! Lütfen tekrar deneyin.",
		"name":                 "İsim",
		"size":                 "Boyut",
		"modified":             "Son Değişiklik",
		"actions":              "İşlemler",
		"download":             "İndir",
		"download_zip":         "Tümünü ZIP İndir",
		"folder_empty":         "Bu klasör boş",
		"parent_dir":           "Üst Klasöre Git",
		"upload_title":         "Dosya Yükle",
		"upload_hint":          "Dosyaları buraya sürükleyin veya seçin",
		"upload_btn":           "Yükle",
		"upload_success":       "Dosya başarıyla yüklendi!",
		"upload_disabled":      "Yükleme modu devre dışı (salt okunur).",
		"tunnel_starting":      "Cloudflare Quick Tunnel başlatılıyor (0 TL, sunucusuz)...",
		"tunnel_ready":         "Tünel hazır! Karşı taraf bu linkle her yerden erişebilir:",
		"tunnel_failed":        "Tünel başlatılamadı. Doğrudan yerel LAN modu kullanılacak.",
		"press_ctrl_c":         "Paylaşımı durdurmak için Ctrl+C tuşlarına basın.",
		"single_file_mode":     "Tek dosya paylaşım modu aktif.",
		"quick_drop_success":   "Dosya başarıyla indirildi.",
		"search_placeholder":   "Dosyalarda ara...",
		"view_list":            "Liste",
		"view_grid":            "Izgara",
		"preview":              "Önizle",
		"close":                "Kapat",
		"copy":                 "Kopyala",
		"copied":               "Kopyalandı!",
		"qr_title":             "Mobil veya Diğer Cihaz İçin QR Kod",

		// Host Dashboard
		"host_title":           "folder-together Yönetici Paneli",
		"host_subtitle":        "Paylaşım Yapılandırması & Canlı İzleme",
		"tab_folder":           "📁 Klasör Paylaş",
		"tab_file":             "📄 Tek Dosya Paylaş (Quick Drop)",
		"browse_btn":           "Gözat...",
		"path_placeholder":     "Paylaşılacak yol...",
		"sec_options":          "Güvenlik & Seçenekler",
		"opt_upload":           "Karşı tarafın dosya yüklemesine izin ver (Upload)",
		"opt_tunnel":           "İnternet Tüneli Aç (Farklı ağlardan erişim için)",
		"pin_mode_none":        "PIN Yok (Açık)",
		"pin_mode_auto":        "Rastgele PIN (4 Haneli)",
		"pin_mode_custom":      "Özel PIN Belirle",
		"btn_start_share":      "▶️ Paylaşımı Başlat",
		"btn_stop_share":       "⏹️ Paylaşımı Durdur",
		"links_title":          "Bağlantı Bilgileri (Karşı Tarafa İletin)",
		"activity_title":       "Canlı Aktivite Günlüğü",
		"no_activity":          "Henüz bir aktivite yok.",
		"selected_path":        "Seçilen Yol",
	},
	LangEN: {
		// App & General
		"app_title":            "folder-together",
		"app_desc":             "Fast & Secure File Sharing between Linux & Windows",
		"status_active":        "Share Active",
		"status_stopped":       "Share Stopped",
		"sharing_folder":       "Shared Folder",
		"sharing_file":         "Shared File",
		"local_url":            "Local Network (LAN) Address",
		"tunnel_url":           "Internet / Tunnel Address (Cross-Network)",
		"pin_code":             "Access PIN Code",
		"pin_required":         "A PIN code is required to access this share",
		"pin_placeholder":      "Enter PIN code",
		"pin_submit":           "Unlock",
		"pin_invalid":          "Invalid PIN code! Please try again.",
		"name":                 "Name",
		"size":                 "Size",
		"modified":             "Last Modified",
		"actions":              "Actions",
		"download":             "Download",
		"download_zip":         "Download All as ZIP",
		"folder_empty":         "This folder is empty",
		"parent_dir":           "Go to Parent Directory",
		"upload_title":         "Upload Files",
		"upload_hint":          "Drag & drop files here or select",
		"upload_btn":           "Upload",
		"upload_success":       "File uploaded successfully!",
		"upload_disabled":      "Upload mode is disabled (read-only).",
		"tunnel_starting":      "Starting Cloudflare Quick Tunnel (0 cost, serverless)...",
		"tunnel_ready":         "Tunnel ready! The peer can access from anywhere using this link:",
		"tunnel_failed":        "Failed to start tunnel. Using direct local LAN mode only.",
		"press_ctrl_c":         "Press Ctrl+C to stop sharing.",
		"single_file_mode":     "Single file quick share mode active.",
		"quick_drop_success":   "File downloaded successfully.",
		"search_placeholder":   "Search files...",
		"view_list":            "List",
		"view_grid":            "Grid",
		"preview":              "Preview",
		"close":                "Close",
		"copy":                 "Copy",
		"copied":               "Copied!",
		"qr_title":             "QR Code for Mobile or Peer Device",

		// Host Dashboard
		"host_title":           "folder-together Host Dashboard",
		"host_subtitle":        "Share Configuration & Live Monitoring",
		"tab_folder":           "📁 Share Folder",
		"tab_file":             "📄 Share Single File (Quick Drop)",
		"browse_btn":           "Browse...",
		"path_placeholder":     "Path to share...",
		"sec_options":          "Security & Options",
		"opt_upload":           "Allow peer to upload files",
		"opt_tunnel":           "Enable Internet Tunnel (for cross-network access)",
		"pin_mode_none":        "No PIN (Open)",
		"pin_mode_auto":        "Random PIN (4 Digits)",
		"pin_mode_custom":      "Custom PIN",
		"btn_start_share":      "▶️ Start Sharing",
		"btn_stop_share":       "⏹️ Stop Sharing",
		"links_title":          "Access Links (Send to Peer)",
		"activity_title":       "Live Activity Log",
		"no_activity":          "No activity yet.",
		"selected_path":        "Selected Path",
	},
}

// SetLanguage sets the active language ('tr' or 'en')
func SetLanguage(lang string) {
	clean := strings.ToLower(strings.TrimSpace(lang))
	if clean == "en" {
		currentLang = LangEN
	} else {
		currentLang = LangTR
	}
}

// AutoDetectLanguage checks LANG/LC_ALL environment variables
func AutoDetectLanguage() {
	envLang := os.Getenv("LANG")
	if envLang == "" {
		envLang = os.Getenv("LC_ALL")
	}
	if strings.HasPrefix(strings.ToLower(envLang), "tr") {
		currentLang = LangTR
	} else if envLang != "" {
		currentLang = LangEN
	}
}

// GetLang returns current language code
func GetLang() string {
	return string(currentLang)
}

// T translates a key into the active language
func T(key string) string {
	if dict, ok := translations[currentLang]; ok {
		if val, exists := dict[key]; exists {
			return val
		}
	}
	if dict, ok := translations[LangEN]; ok {
		if val, exists := dict[key]; exists {
			return val
		}
	}
	return key
}

// GetAllTranslations returns the full map for client injection
func GetAllTranslations() map[string]string {
	return translations[currentLang]
}
