//go:build windows

package tunnel

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"os"
	"path/filepath"
)

//go:embed embeds/cloudflared-windows.exe.gz
var embeddedCloudflaredGz []byte

func extractEmbeddedCloudflared() (string, error) {
	destDir := os.Getenv("LOCALAPPDATA")
	if destDir == "" {
		destDir = os.TempDir()
	}
	appDir := filepath.Join(destDir, "folder-together")
	_ = os.MkdirAll(appDir, 0755)

	targetExe := filepath.Join(appDir, "cloudflared.exe")

	// If already extracted with valid size (>1MB), reuse it immediately
	if fi, err := os.Stat(targetExe); err == nil && fi.Size() > 1024*1024 {
		return targetExe, nil
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(embeddedCloudflaredGz))
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tmpExe := targetExe + ".tmp"
	out, err := os.OpenFile(tmpExe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(out, gzReader); err != nil {
		out.Close()
		_ = os.Remove(tmpExe)
		return "", err
	}
	out.Close()

	if err := os.Rename(tmpExe, targetExe); err != nil {
		return tmpExe, nil
	}

	return targetExe, nil
}
