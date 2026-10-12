//go:build linux

package tunnel

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed embeds/cloudflared-linux.gz
var embeddedCloudflaredGz []byte

func extractEmbeddedCloudflared() (string, error) {
	var candidateDirs []string

	// 1. Same directory as the running folder-together executable
	if exePath, err := os.Executable(); err == nil {
		candidateDirs = append(candidateDirs, filepath.Dir(exePath))
	}

	// 2. ~/.local/bin
	if homeDir, err := os.UserHomeDir(); err == nil {
		candidateDirs = append(candidateDirs, filepath.Join(homeDir, ".local", "bin"))
	}

	// 3. /tmp/folder-together
	candidateDirs = append(candidateDirs, filepath.Join(os.TempDir(), "folder-together"))

	// 4. Current working directory
	candidateDirs = append(candidateDirs, ".")

	// Phase 1: Check if already extracted (>30MB)
	for _, dir := range candidateDirs {
		targetBin := filepath.Join(dir, "cloudflared")
		if fi, err := os.Stat(targetBin); err == nil && fi.Size() > 30*1024*1024 {
			_ = os.Chmod(targetBin, 0755)
			return targetBin, nil
		}
	}

	// Phase 2: Extract to first writable candidate
	var lastErr error
	for _, dir := range candidateDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			lastErr = err
			continue
		}

		targetBin := filepath.Join(dir, "cloudflared")
		_ = os.Remove(targetBin)

		out, err := os.OpenFile(targetBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			targetBin = filepath.Join(dir, fmt.Sprintf("cloudflared_%d", os.Getpid()))
			_ = os.Remove(targetBin)
			out, err = os.OpenFile(targetBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				lastErr = err
				continue
			}
		}

		gzReader, err := gzip.NewReader(bytes.NewReader(embeddedCloudflaredGz))
		if err != nil {
			out.Close()
			_ = os.Remove(targetBin)
			return "", fmt.Errorf("embedded gzip parse error: %w", err)
		}

		written, copyErr := io.Copy(out, gzReader)
		gzReader.Close()
		out.Close()

		if copyErr != nil || written < 30*1024*1024 {
			_ = os.Remove(targetBin)
			lastErr = fmt.Errorf("extraction incomplete (written %d bytes): %v", written, copyErr)
			continue
		}

		_ = os.Chmod(targetBin, 0755)
		return targetBin, nil
	}

	return "", fmt.Errorf("failed to extract cloudflared to any candidate directory: %w", lastErr)
}
