package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"folder-together/internal/host"
	"folder-together/internal/i18n"
	"folder-together/internal/server"
	"folder-together/internal/tunnel"
)

func generatePIN(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		b[i] = digits[num.Int64()]
	}
	return string(b)
}

func findFreePort(startPort int) int {
	for p := startPort; p < startPort+100; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			ln.Close()
			return p
		}
	}
	return startPort
}

func main() {
	dirFlag := flag.String("dir", "", "Klasör yolu / Folder path to share")
	fileFlag := flag.String("file", "", "Tek dosya paylaşımı / Single file to share")
	portFlag := flag.Int("port", 0, "Port numarası / Port (default: 8080 or auto)")
	pinFlag := flag.String("pin", "", "Erişim PIN kodu / Access PIN ('auto' for random 4 digits)")
	uploadFlag := flag.Bool("upload", false, "Karşı tarafın dosya yüklemesine izin ver / Allow peer to upload")
	noTunnelFlag := flag.Bool("no-tunnel", false, "Tüneli devre dışı bırak, sadece LAN / Disable tunnel, LAN only")
	noBrowserFlag := flag.Bool("no-browser", false, "Yönetici arayüzü için tarayıcıyı otomatik açma / Don't open browser")
	guiFlag := flag.Bool("gui", false, "Masaüstü Yönetici Arayüzü modunda başlat / Start in GUI dashboard mode")
	langFlag := flag.String("lang", "", "Dil seçimi / Language ('tr' veya 'en')")
	flag.Parse()

	// Language detection
	if *langFlag != "" {
		i18n.SetLanguage(*langFlag)
	} else {
		i18n.AutoDetectLanguage()
	}

	// Port determination
	listenPort := *portFlag
	if listenPort == 0 {
		listenPort = findFreePort(8080)
	}

	// Check if started in pure GUI mode (no dir and no file specified, or -gui explicitly requested)
	guiMode := *guiFlag || (*dirFlag == "" && *fileFlag == "")

	singleFileMode := false
	var targetPath string
	autoStart := false

	if *fileFlag != "" {
		absFile, err := filepath.Abs(*fileFlag)
		if err != nil {
			fmt.Printf("Geçersiz dosya yolu: %v\n", err)
			os.Exit(1)
		}
		singleFileMode = true
		targetPath = absFile
		autoStart = true
	} else if *dirFlag != "" {
		absDir, err := filepath.Abs(*dirFlag)
		if err != nil {
			fmt.Printf("Geçersiz klasör yolu: %v\n", err)
			os.Exit(1)
		}
		targetPath = absDir
		autoStart = true
	} else {
		// Default path is current directory, always auto-start sharing!
		cwd, _ := os.Getwd()
		targetPath = cwd
		autoStart = true
	}

	// PIN Code
	activePIN := ""
	if *pinFlag == "auto" {
		activePIN = generatePIN(4)
	} else if *pinFlag != "" {
		activePIN = *pinFlag
	}

	srvConfig := server.Config{
		RootDir:        targetPath,
		Port:           listenPort,
		PIN:            activePIN,
		AllowUpload:    *uploadFlag,
		SingleFile:     singleFileMode,
		SingleFilePath: targetPath,
		AutoStart:      autoStart,
	}

	srv := server.New(srvConfig)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in background
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf("Server hatası: %v\n", err)
			os.Exit(1)
		}
	}()

	hostDashboardURL := fmt.Sprintf("http://127.0.0.1:%d/host", listenPort)

	// In GUI mode or by default, open the desktop host dashboard
	if !*noBrowserFlag {
		go func() {
			time.Sleep(300 * time.Millisecond)
			_ = host.OpenBrowser(hostDashboardURL)
		}()
	}

	// Banner display
	fmt.Println("\n==================================================================")
	fmt.Printf("  📁 %s - %s\n", i18n.T("app_title"), i18n.T("app_desc"))
	fmt.Println("==================================================================")
	fmt.Printf("  ⚡ Masaüstü Host Yönetici Paneli: %s\n", hostDashboardURL)

	var tunnelCleanup func()

	if autoStart {
		fmt.Printf("  📂 %s: %s\n", i18n.T("sharing_folder"), targetPath)
		if activePIN != "" {
			fmt.Printf("  🔑 %s: %s\n", i18n.T("pin_code"), activePIN)
		}

		// LAN IPs
		localIPs := tunnel.GetLocalIPs()
		fmt.Printf("\n  🌐 %s:\n", i18n.T("local_url"))
		for _, ip := range localIPs {
			fmt.Printf("     👉 http://%s:%d\n", ip, listenPort)
		}

		// Quick Tunnel
		if !*noTunnelFlag {
			fmt.Printf("\n  🚀 %s\n", i18n.T("tunnel_starting"))
			publicURL, cleanup, err := tunnel.StartQuickTunnel(ctx, listenPort)
			if err != nil {
				fmt.Printf("  ⚠️  %s (%v)\n", i18n.T("tunnel_failed"), err)
			} else {
				tunnelCleanup = cleanup
				srv.SetTunnelURL(publicURL, cleanup)
				fmt.Printf("  ✨ %s\n", i18n.T("tunnel_ready"))
				fmt.Printf("     👉 %s\n", publicURL)
			}
		}
	} else if guiMode {
		fmt.Println("  💡 Masaüstü Kontrol Arayüzü açıldı. Paylaşımı oradan başlatabilirsiniz.")
	}

	fmt.Println("\n------------------------------------------------------------------")
	fmt.Printf("  🛑 %s\n", i18n.T("press_ctrl_c"))
	fmt.Println("==================================================================")

	// Wait for shutdown
	<-sigChan
	fmt.Println("\nKapatılıyor / Shutting down...")
	if tunnelCleanup != nil {
		tunnelCleanup()
	}
	cancel()
}
