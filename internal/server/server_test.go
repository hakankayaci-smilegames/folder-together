package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadAndNavigation(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "ft_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subfolder and test files
	subDir := filepath.Join(tempDir, "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subDir: %v", err)
	}

	testFile := filepath.Join(tempDir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := Config{
		RootDir:     tempDir,
		Port:        9999,
		PIN:         "1234",
		AllowUpload: true,
		AutoStart:   true,
	}
	srv := New(cfg)

	// 1. Test /api/list without PIN -> should fail 401
	{
		req := httptest.NewRequest(http.MethodGet, "/api/list?path=/", nil)
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleList)(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 Unauthorized for list without PIN, got %d", rec.Code)
		}
	}

	// 2. Test /api/list with valid PIN -> should succeed 200 and return hello.txt and subfolder
	{
		req := httptest.NewRequest(http.MethodGet, "/api/list?path=/", nil)
		req.Header.Set("X-PIN", "1234")
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleList)(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for list with PIN, got %d", rec.Code)
		}

		var items []FileItem
		if err := json.NewDecoder(rec.Body).Decode(&items); err != nil {
			t.Fatalf("Failed to decode items: %v", err)
		}
		if len(items) != 2 {
			t.Errorf("Expected 2 items in root, got %d", len(items))
		}
	}

	// 3. Test /api/list for subfolder
	{
		req := httptest.NewRequest(http.MethodGet, "/api/list?path=/subfolder", nil)
		req.Header.Set("X-PIN", "1234")
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleList)(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for list subfolder, got %d", rec.Code)
		}
	}

	// 4. Test Upload to Root with valid PIN
	{
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("path", "/")
		part, err := writer.CreateFormFile("files", "uploaded_root.txt")
		if err != nil {
			t.Fatalf("CreateFormFile error: %v", err)
		}
		_, _ = io.WriteString(part, "file content uploaded to root")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-PIN", "1234")
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleUpload)(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Upload to root failed: status %d, body %s", rec.Code, rec.Body.String())
		}

		// Verify file exists on disk
		uploadedPath := filepath.Join(tempDir, "uploaded_root.txt")
		content, err := os.ReadFile(uploadedPath)
		if err != nil {
			t.Fatalf("Uploaded file not found on disk: %v", err)
		}
		if string(content) != "file content uploaded to root" {
			t.Errorf("Uploaded content mismatch: got %q", string(content))
		}
	}

	// 5. Test Upload to Subfolder with valid PIN
	{
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("path", "/subfolder")
		part, err := writer.CreateFormFile("files", "uploaded_sub.txt")
		if err != nil {
			t.Fatalf("CreateFormFile error: %v", err)
		}
		_, _ = io.WriteString(part, "file content uploaded to subfolder")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-PIN", "1234")
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleUpload)(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Upload to subfolder failed: status %d, body %s", rec.Code, rec.Body.String())
		}

		// Verify file exists on disk inside subfolder
		uploadedSubPath := filepath.Join(subDir, "uploaded_sub.txt")
		content, err := os.ReadFile(uploadedSubPath)
		if err != nil {
			t.Fatalf("Uploaded subfolder file not found: %v", err)
		}
		if string(content) != "file content uploaded to subfolder" {
			t.Errorf("Uploaded subfolder content mismatch: got %q", string(content))
		}
	}

	// 6. Test Upload when AllowUpload is false -> should fail 403
	{
		srv.mu.Lock()
		srv.cfg.AllowUpload = false
		srv.mu.Unlock()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("path", "/")
		part, _ := writer.CreateFormFile("files", "disallowed.txt")
		_, _ = io.WriteString(part, "should fail")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-PIN", "1234")
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleUpload)(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden when upload disabled, got %d", rec.Code)
		}
	}

	// 7. Test Path Traversal Upload attempt -> should fail 403
	{
		srv.mu.Lock()
		srv.cfg.AllowUpload = true
		srv.mu.Unlock()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("path", "../../etc")
		part, _ := writer.CreateFormFile("files", "hack.txt")
		_, _ = io.WriteString(part, "should fail")
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-PIN", "1234")
		rec := httptest.NewRecorder()
		srv.authMiddleware(srv.handleUpload)(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden for path traversal, got %d", rec.Code)
		}
	}
}
