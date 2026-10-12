package tunnel

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedWindowsFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("embeds", "cloudflared-windows.exe.gz"))
	if err != nil {
		t.Fatalf("Failed to read embeds/cloudflared-windows.exe.gz: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("embeds/cloudflared-windows.exe.gz is empty!")
	}
	t.Logf("Embedded Windows gz size on disk: %d bytes", len(data))

	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	header := make([]byte, 2)
	if _, err := io.ReadFull(gzReader, header); err != nil {
		t.Fatalf("Failed to read header: %v", err)
	}
	// PE header: 'M' 'Z'
	if header[0] != 'M' || header[1] != 'Z' {
		t.Fatalf("Invalid PE header: %v", header)
	}
	t.Logf("PE header verified successfully!")
}

func TestEmbeddedLinuxGz(t *testing.T) {
	if len(embeddedCloudflaredGz) == 0 {
		t.Fatalf("embeddedCloudflaredGz is empty!")
	}
	t.Logf("Embedded Linux gz size: %d bytes", len(embeddedCloudflaredGz))

	gzReader, err := gzip.NewReader(bytes.NewReader(embeddedCloudflaredGz))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	header := make([]byte, 4)
	if _, err := io.ReadFull(gzReader, header); err != nil {
		t.Fatalf("Failed to read header: %v", err)
	}
	// ELF header: 0x7f 'E' 'L' 'F'
	if header[0] != 0x7f || header[1] != 'E' || header[2] != 'L' || header[3] != 'F' {
		t.Fatalf("Invalid ELF header: %v", header)
	}
	t.Logf("ELF header verified successfully!")
}

func TestExtractEmbeddedCloudflaredLinux(t *testing.T) {
	binPath, err := extractEmbeddedCloudflared()
	if err != nil {
		t.Fatalf("extractEmbeddedCloudflared failed: %v", err)
	}
	fi, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("Extracted file does not exist: %v", err)
	}
	if fi.Size() < 30*1024*1024 {
		t.Fatalf("Extracted file size too small: %d bytes", fi.Size())
	}
	t.Logf("Successfully extracted cloudflared to: %s (%d bytes)", binPath, fi.Size())
}
