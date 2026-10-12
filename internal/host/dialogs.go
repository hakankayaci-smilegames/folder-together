package host

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PickFolder launches the native OS folder picker and returns the selected path
func PickFolder() (string, error) {
	switch runtime.GOOS {
	case "linux":
		// Check for KDE kdialog first, then zenity
		if _, err := exec.LookPath("kdialog"); err == nil {
			cmd := exec.Command("kdialog", "--getexistingdirectory")
			out, err := cmd.Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
		if _, err := exec.LookPath("zenity"); err == nil {
			cmd := exec.Command("zenity", "--file-selection", "--directory", "--title=Paylaşılacak Klasörü Seçin")
			out, err := cmd.Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
		return "", errors.New("kdialog veya zenity bulunamadı")

	case "windows":
		psScript := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object Windows.Forms.FolderBrowserDialog; $f.Description = 'Paylaşılacak Klasörü Seçin'; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $f.SelectedPath }`
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil

	default:
		return "", errors.New("desteklenmeyen işletim sistemi")
	}
}

// PickFile launches the native OS file picker and returns the selected path
func PickFile() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("kdialog"); err == nil {
			cmd := exec.Command("kdialog", "--getopenfilename")
			out, err := cmd.Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
		if _, err := exec.LookPath("zenity"); err == nil {
			cmd := exec.Command("zenity", "--file-selection", "--title=Paylaşılacak Dosyayı Seçin")
			out, err := cmd.Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
		return "", errors.New("kdialog veya zenity bulunamadı")

	case "windows":
		psScript := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object Windows.Forms.OpenFileDialog; $f.Title = 'Paylaşılacak Dosyayı Seçin'; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { Write-Output $f.FileName }`
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil

	default:
		return "", errors.New("desteklenmeyen işletim sistemi")
	}
}

// OpenBrowser opens the URL in the user's default browser
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return errors.New("tarayıcı açılamadı")
	}
}

// GetUserHomeDir returns the user home directory safely
func GetUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// CleanOutput sanitizes command output
func clean(b []byte) string {
	return string(bytes.TrimSpace(b))
}
