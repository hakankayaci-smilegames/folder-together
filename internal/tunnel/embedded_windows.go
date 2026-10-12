//go:build windows

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

//go:embed embeds/cloudflared-windows.exe.gz
var embeddedCloudflaredGz []byte

func extractEmbeddedCloudflared() (string, error) {
	var candidateDirs []string

	// 1. Same directory as the running folder-together executable (highest priority & native execution rights)
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidateDirs = append(candidateDirs, exeDir)
	}

	// 2. %LOCALAPPDATA%\folder-together
	if localApp := os.Getenv("LOCALAPPDATA"); localApp != "" {
		candidateDirs = append(candidateDirs, filepath.Join(localApp, "folder-together"))
	}

	// 3. %USERPROFILE%\.folder-together
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		candidateDirs = append(candidateDirs, filepath.Join(userProfile, ".folder-together"))
	}

	// 4. %TEMP%\folder-together
	if tempDir := os.TempDir(); tempDir != "" {
		candidateDirs = append(candidateDirs, filepath.Join(tempDir, "folder-together"))
	}

	// 5. Current working directory
	candidateDirs = append(candidateDirs, ".")

	// Phase 1: Check if already extracted and valid (>30MB)
	for _, dir := range candidateDirs {
		targetExe := filepath.Join(dir, "cloudflared.exe")
		if fi, err := os.Stat(targetExe); err == nil && fi.Size() > 30*1024*1024 {
			return targetExe, nil
		}
	}

	// Phase 2: Extract to the first writable directory
	var lastErr error
	for _, dir := range candidateDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			lastErr = err
			continue
		}

		targetExe := filepath.Join(dir, "cloudflared.exe")

		// Remove any stale 0-byte or corrupted file
		_ = os.Remove(targetExe)

		// Direct write to cloudflared.exe (avoids .tmp rename locking issues)
		out, err := os.OpenFile(targetExe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			// If cloudflared.exe is locked, try PID-specific name ending in .exe
			targetExe = filepath.Join(dir, fmt.Sprintf("cloudflared_%d.exe", os.Getpid()))
			_ = os.Remove(targetExe)
			out, err = os.OpenFile(targetExe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				lastErr = err
				continue
			}
		}

		gzReader, err := gzip.NewReader(bytes.NewReader(embeddedCloudflaredGz))
		if err != nil {
			out.Close()
			_ = os.Remove(targetExe)
			return "", fmt.Errorf("embedded gzip parse error: %w", err)
		}

		written, copyErr := io.Copy(out, gzReader)
		gzReader.Close()
		out.Close()

		if copyErr != nil || written < 30*1024*1024 {
			_ = os.Remove(targetExe)
			lastErr = fmt.Errorf("extraction incomplete (written %d bytes): %v", written, copyErr)
			continue
		}

		return targetExe, nil
	}

	return "", fmt.Errorf("failed to extract cloudflared to any candidate directory: %w", lastErr)
}
