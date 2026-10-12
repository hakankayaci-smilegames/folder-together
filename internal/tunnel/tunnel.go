package tunnel

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"folder-together/internal/i18n"
)

var tunnelURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

// GetLocalIPs returns non-loopback IPv4 addresses of the host machine
func GetLocalIPs() []string {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // skip IPv6 for simplicity
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}

// FindOrDownloadCloudflared ensures cloudflared binary is available
func FindOrDownloadCloudflared() (string, error) {
	binaryName := "cloudflared"
	if runtime.GOOS == "windows" {
		binaryName = "cloudflared.exe"
	}

	// 1. Check in PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path, nil
	}

	// 2. Check in current executable directory or ~/.local/bin
	homeDir, _ := os.UserHomeDir()
	candidatePaths := []string{
		binaryName,
		filepath.Join(".", binaryName),
		filepath.Join(homeDir, ".local", "bin", binaryName),
	}
	for _, p := range candidatePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// 3. Attempt download
	targetPath := filepath.Join(".", binaryName)
	downloadURL := ""
	if runtime.GOOS == "windows" {
		downloadURL = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe"
	} else if runtime.GOOS == "linux" {
		downloadURL = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64"
	} else {
		return "", fmt.Errorf("unsupported OS for auto-download: %s", runtime.GOOS)
	}

	fmt.Printf("cloudflared bulunamadı, indiriliyor (bir seferlik): %s\n", downloadURL)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	return targetPath, nil
}

// StartQuickTunnel launches cloudflared quick tunnel and parses the public trycloudflare.com URL
func StartQuickTunnel(ctx context.Context, localPort int) (string, func(), error) {
	cfBin, err := FindOrDownloadCloudflared()
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", i18n.T("tunnel_failed"), err)
	}

	localTarget := fmt.Sprintf("http://127.0.0.1:%d", localPort)
	// Using --protocol http2 guarantees firewall bypass (TCP 443) instead of UDP QUIC which is frequently blocked
	cmd := exec.CommandContext(ctx, cfBin, "tunnel", "--protocol", "http2", "--url", localTarget, "--no-autoupdate")

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", nil, err
	}

	if err := cmd.Start(); err != nil {
		return "", nil, err
	}

	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := tunnelURLRegex.FindString(line); match != "" {
				urlChan <- match
				break
			}
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	select {
	case url := <-urlChan:
		cleanup := func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
		return url, cleanup, nil
	case err := <-errChan:
		_ = cmd.Process.Kill()
		return "", nil, err
	case <-time.After(35 * time.Second):
		_ = cmd.Process.Kill()
		return "", nil, fmt.Errorf("tunnel timeout after 35 seconds")
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return "", nil, ctx.Err()
	}
}

// ShortenURL generates a clean, short URL using free APIs (ulvis.net, clck.ru fallback)
func ShortenURL(longURL string) string {
	if longURL == "" {
		return ""
	}
	client := &http.Client{Timeout: 4 * time.Second}

	// 1. Try ulvis.net (clean 301 redirect, no captcha, supports trycloudflare)
	reqURL := fmt.Sprintf("https://ulvis.net/API/write/get?url=%s&type=json", url.QueryEscape(longURL))
	resp, err := client.Get(reqURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var res struct {
			Success bool `json:"success"`
			Data    struct {
				URL string `json:"url"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Success && res.Data.URL != "" {
			return res.Data.URL
		}
	}

	// 2. Fallback to clck.ru
	clckURL := fmt.Sprintf("https://clck.ru/--?url=%s", url.QueryEscape(longURL))
	resp2, err2 := client.Get(clckURL)
	if err2 == nil && resp2.StatusCode == http.StatusOK {
		defer resp2.Body.Close()
		b, _ := io.ReadAll(resp2.Body)
		s := strings.TrimSpace(string(b))
		if strings.HasPrefix(s, "http") {
			return s
		}
	}

	return ""
}
