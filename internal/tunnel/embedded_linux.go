//go:build linux

package tunnel

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"os"
	"path/filepath"
)

//go:embed embeds/cloudflared-linux.gz
var embeddedCloudflaredGz []byte

func extractEmbeddedCloudflared() (string, error) {
	homeDir, _ := os.UserHomeDir()
	appDir := filepath.Join(homeDir, ".local", "bin")
	_ = os.MkdirAll(appDir, 0755)

	targetBin := filepath.Join(appDir, "cloudflared")

	// If already extracted with valid size (>1MB), reuse it
	if fi, err := os.Stat(targetBin); err == nil && fi.Size() > 1024*1024 {
		return targetBin, nil
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(embeddedCloudflaredGz))
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tmpBin := filepath.Join(appDir, "cloudflared.tmp")
	out, err := os.OpenFile(tmpBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(out, gzReader); err != nil {
		out.Close()
		_ = os.Remove(tmpBin)
		return "", err
	}
	out.Close()

	_ = os.Rename(tmpBin, targetBin)
	_ = os.Chmod(targetBin, 0755)

	return targetBin, nil
}
