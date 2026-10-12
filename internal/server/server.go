package server

import (
	"archive/zip"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"folder-together/internal/host"
	"folder-together/internal/i18n"
	"folder-together/internal/tunnel"
)

//go:embed web/*
var webFS embed.FS

type ActivityEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // "download", "upload", "browse", "system"
}

type Config struct {
	RootDir        string
	Port           int
	PIN            string
	AllowUpload    bool
	SingleFile     bool
	SingleFilePath string
	AutoStart      bool
}

type FileItem struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

type HostStartRequest struct {
	Path         string `json:"path"`
	IsSingleFile bool   `json:"is_single_file"`
	PIN          string `json:"pin"`
	AllowUpload  bool   `json:"allow_upload"`
	EnableTunnel bool   `json:"enable_tunnel"`
}

type Server struct {
	mu           sync.RWMutex
	cfg          Config
	isRunning    bool
	tunnelURL    string
	shortURL     string
	tunnelCancel context.CancelFunc
	events       []ActivityEvent
	httpServer   *http.Server
}

func New(cfg Config) *Server {
	s := &Server{
		cfg:       cfg,
		isRunning: cfg.AutoStart,
		events:    make([]ActivityEvent, 0),
	}
	if cfg.AutoStart {
		s.logEventLocked("Paylaşım başlatıldı", "system")
	}
	return s
}

func (s *Server) logEventLocked(msg string, eventType string) {
	s.events = append(s.events, ActivityEvent{
		Timestamp: time.Now(),
		Message:   msg,
		Type:      eventType,
	})
	// Keep last 50 events
	if len(s.events) > 50 {
		s.events = s.events[len(s.events)-50:]
	}
}

func (s *Server) LogEvent(msg string, eventType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logEventLocked(msg, eventType)
}

func (s *Server) SetTunnelURL(url string, shortURL string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tunnelURL = url
	s.shortURL = shortURL
	s.tunnelCancel = cancel
	if shortURL != "" {
		s.logEventLocked(fmt.Sprintf("İnternet tüneli açıldı: %s (Kısa Link: %s)", url, shortURL), "system")
	} else {
		s.logEventLocked(fmt.Sprintf("İnternet tüneli açıldı: %s", url), "system")
	}
}

func (s *Server) GetPort() int {
	return s.cfg.Port
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	webSub, err := fs.Sub(webFS, "web")
	if err != nil {
		return err
	}

	// Guest UI & static assets
	mux.HandleFunc("/", s.handleRoot(webSub))
	mux.HandleFunc("/host", s.handleHostUI(webSub))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webSub))))

	// Guest APIs
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/i18n", s.handleI18n)
	mux.HandleFunc("/api/list", s.authMiddleware(s.handleList))
	mux.HandleFunc("/api/download", s.authMiddleware(s.handleDownload))
	mux.HandleFunc("/api/preview", s.authMiddleware(s.handlePreview))
	mux.HandleFunc("/api/zip", s.authMiddleware(s.handleZip))
	mux.HandleFunc("/api/upload", s.authMiddleware(s.handleUpload))

	// Host Management APIs (Only accessible from localhost!)
	mux.HandleFunc("/api/host/status", s.localOnly(s.handleHostStatus))
	mux.HandleFunc("/api/host/browse", s.localOnly(s.handleHostBrowse))
	mux.HandleFunc("/api/host/start", s.localOnly(s.handleHostStart))
	mux.HandleFunc("/api/host/stop", s.localOnly(s.handleHostStop))
	mux.HandleFunc("/api/host/events", s.localOnly(s.handleHostEvents))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

// localOnly restricts management APIs to localhost
func (s *Server) localOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		ip := net.ParseIP(host)
		if ip == nil || (!ip.IsLoopback() && host != "localhost") {
			http.Error(w, "Access forbidden: host management is localhost only", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		running := s.isRunning
		pin := s.cfg.PIN
		s.mu.RUnlock()

		if !running {
			http.Error(w, i18n.T("status_stopped"), http.StatusServiceUnavailable)
			return
		}

		if pin != "" {
			providedPin := r.Header.Get("X-PIN")
			if providedPin == "" {
				providedPin = r.URL.Query().Get("pin")
			}
			if providedPin != pin {
				http.Error(w, i18n.T("pin_invalid"), http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleRoot(webSub fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		content, err := fs.ReadFile(webSub, "index.html")
		if err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	}
}

func (s *Server) handleHostUI(webSub fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify local access
		hostIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			hostIP = r.RemoteAddr
		}
		ip := net.ParseIP(hostIP)
		if ip == nil || (!ip.IsLoopback() && hostIP != "localhost") {
			http.Error(w, "Access forbidden: host panel is localhost only", http.StatusForbidden)
			return
		}

		content, err := fs.ReadFile(webSub, "host.html")
		if err != nil {
			http.Error(w, "Host template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(content)
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := map[string]interface{}{
		"uploadEnabled": s.cfg.AllowUpload,
		"pinRequired":   s.cfg.PIN != "",
		"singleFile":    s.cfg.SingleFile,
		"targetName":    filepath.Base(s.cfg.RootDir),
		"isRunning":     s.isRunning,
		"lang":          i18n.GetLang(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleI18n(w http.ResponseWriter, r *http.Request) {
	langReq := r.URL.Query().Get("lang")
	if langReq != "" {
		i18n.SetLanguage(langReq)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(i18n.GetAllTranslations())
}

func (s *Server) resolveSafePath(reqPath string) (string, error) {
	s.mu.RLock()
	baseRoot := s.cfg.RootDir
	s.mu.RUnlock()

	cleanRel := filepath.Clean(filepath.FromSlash(reqPath))
	if cleanRel == "." || cleanRel == "/" || cleanRel == "\\" {
		return baseRoot, nil
	}
	cleanRel = strings.TrimPrefix(cleanRel, string(filepath.Separator))
	cleanRel = strings.TrimPrefix(cleanRel, "/")
	cleanRel = strings.TrimPrefix(cleanRel, "\\")

	full := filepath.Join(baseRoot, cleanRel)
	rel, err := filepath.Rel(baseRoot, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("access denied: path traversal detected")
	}
	return full, nil
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	isSingle := s.cfg.SingleFile
	singlePath := s.cfg.SingleFilePath
	s.mu.RUnlock()

	if isSingle {
		fi, err := os.Stat(singlePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items := []FileItem{{
			Name:    filepath.Base(singlePath),
			IsDir:   false,
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
		}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "/"
	}

	targetDir, err := s.resolveSafePath(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, FileItem{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	s.LogEvent(fmt.Sprintf("Dizin görüntülendi: %s (%d öge)", reqPath, len(items)), "browse")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	isSingle := s.cfg.SingleFile
	singlePath := s.cfg.SingleFilePath
	s.mu.RUnlock()

	if isSingle {
		s.LogEvent(fmt.Sprintf("Dosya indirildi: %s", filepath.Base(singlePath)), "download")
		http.ServeFile(w, r, singlePath)
		return
	}

	reqPath := r.URL.Query().Get("path")
	targetFile, err := s.resolveSafePath(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	fi, err := os.Stat(targetFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if fi.IsDir() {
		http.Error(w, "Cannot download folder directly. Use zip.", http.StatusBadRequest)
		return
	}

	s.LogEvent(fmt.Sprintf("Dosya indirildi: %s (%d bytes)", fi.Name(), fi.Size()), "download")
	http.ServeFile(w, r, targetFile)
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	targetFile, err := s.resolveSafePath(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	fi, err := os.Stat(targetFile)
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}

	// For text files up to 2MB, serve with utf-8 text/plain
	ext := strings.ToLower(filepath.Ext(targetFile))
	if ext == ".txt" || ext == ".md" || ext == ".go" || ext == ".py" || ext == ".json" || ext == ".log" || ext == ".yaml" || ext == ".yml" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}

	http.ServeFile(w, r, targetFile)
}

func (s *Server) handleZip(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = "/"
	}

	targetDir, err := s.resolveSafePath(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	folderName := filepath.Base(targetDir)
	if folderName == "." || folderName == "/" || folderName == "\\" {
		folderName = "shared_folder"
	}

	s.LogEvent(fmt.Sprintf("ZIP indiriliyor: %s", folderName), "download")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, folderName))

	zw := zip.NewWriter(w)
	defer zw.Close()

	filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(targetDir, path)
		if err != nil {
			return nil
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		_, err = io.Copy(writer, f)
		return err
	})
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	allowUpload := s.cfg.AllowUpload
	isSingle := s.cfg.SingleFile
	s.mu.RUnlock()

	if !allowUpload {
		http.Error(w, i18n.T("upload_disabled"), http.StatusForbidden)
		return
	}

	if isSingle {
		http.Error(w, "Tek dosya paylaşım modunda yükleme yapılamaz", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(64 << 20)
	if err != nil {
		http.Error(w, "Upload parsing error: "+err.Error(), http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	reqPath := r.FormValue("path")
	targetDir, err := s.resolveSafePath(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	fi, err := os.Stat(targetDir)
	if err != nil || !fi.IsDir() {
		http.Error(w, "Hedef klasör bulunamadı veya geçerli bir klasör değil", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "Yüklenecek dosya bulunamadı", http.StatusBadRequest)
		return
	}

	count := 0
	for _, fh := range files {
		safeFileName := filepath.Base(fh.Filename)
		if safeFileName == "." || safeFileName == "/" || safeFileName == "\\" || safeFileName == "" {
			continue
		}
		dstPath := filepath.Join(targetDir, safeFileName)

		src, err := fh.Open()
		if err != nil {
			continue
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			continue
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()

		if copyErr == nil {
			count++
			s.LogEvent(fmt.Sprintf("Dosya yüklendi: %s (%d bytes)", safeFileName, fh.Size), "upload")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"success":true,"uploaded":%d}`, count)))
}

// Host API Handlers
func (s *Server) handleHostStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := map[string]interface{}{
		"isRunning":    s.isRunning,
		"path":         s.cfg.RootDir,
		"isSingleFile": s.cfg.SingleFile,
		"pin":          s.cfg.PIN,
		"allowUpload":  s.cfg.AllowUpload,
		"port":         s.cfg.Port,
		"localIPs":     tunnel.GetLocalIPs(),
		"tunnelURL":    s.tunnelURL,
		"shortURL":     s.shortURL,
		"events":       s.events,
		"lang":         i18n.GetLang(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleHostBrowse(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode") // "folder" or "file"
	var selected string
	var err error

	if mode == "file" {
		selected, err = host.PickFile()
	} else {
		selected, err = host.PickFolder()
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"path": selected,
	})
}

func (s *Server) handleHostStart(w http.ResponseWriter, r *http.Request) {
	var req HostStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cleanPath, err := filepath.Abs(req.Path)
	if err != nil || cleanPath == "" {
		http.Error(w, "Geçersiz dosya/klasör yolu", http.StatusBadRequest)
		return
	}

	fi, err := os.Stat(cleanPath)
	if err != nil {
		http.Error(w, "Seçilen yol bulunamadı: "+cleanPath, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.cfg.RootDir = cleanPath
	s.cfg.SingleFile = req.IsSingleFile || !fi.IsDir()
	s.cfg.SingleFilePath = cleanPath
	s.cfg.PIN = req.PIN
	s.cfg.AllowUpload = req.AllowUpload
	s.isRunning = true
	s.mu.Unlock()

	s.LogEvent(fmt.Sprintf("Paylaşım başlatıldı: %s", cleanPath), "system")

	// Manage tunnel
	if req.EnableTunnel && s.tunnelURL == "" {
		go func() {
			s.LogEvent("Cloudflare tüneli başlatılıyor...", "system")
			ctx, cancel := context.WithCancel(context.Background())
			pubURL, cleanup, err := tunnel.StartQuickTunnel(ctx, s.cfg.Port)
			if err != nil {
				s.LogEvent("Tünel hatası: "+err.Error(), "system")
				cancel()
			} else {
				shortURL := tunnel.ShortenURL(pubURL)
				s.SetTunnelURL(pubURL, shortURL, func() {
					if cleanup != nil {
						cleanup()
					}
					cancel()
				})
			}
		}()
	} else if !req.EnableTunnel && s.tunnelURL != "" {
		s.mu.Lock()
		if s.tunnelCancel != nil {
			s.tunnelCancel()
			s.tunnelCancel = nil
		}
		s.tunnelURL = ""
		s.shortURL = ""
		s.logEventLocked("Tünel durduruldu.", "system")
		s.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

func (s *Server) handleHostStop(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.isRunning = false
	if s.tunnelCancel != nil {
		s.tunnelCancel()
		s.tunnelCancel = nil
	}
	s.tunnelURL = ""
	s.shortURL = ""
	s.logEventLocked("Paylaşım durduruldu.", "system")
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

func (s *Server) handleHostEvents(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	events := s.events
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
