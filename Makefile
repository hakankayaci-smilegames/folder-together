BINARY_NAME=folder-together

.PHONY: all build build-linux build-windows clean run help

all: build

build: build-linux build-windows

build-linux:
	@mkdir -p bin
	@echo "📦 Linux (Kubuntu) için derleniyor..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-linux cmd/main.go
	@chmod +x bin/$(BINARY_NAME)-linux
	@echo "✅ bin/$(BINARY_NAME)-linux hazır!"

build-windows:
	@mkdir -p bin
	@echo "📦 Windows 10 için derleniyor (.exe)..."
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-windows.exe cmd/main.go
	@echo "✅ bin/$(BINARY_NAME)-windows.exe hazır!"

clean:
	@rm -rf bin
	@echo "Temizlendi."

run:
	@go run cmd/main.go

help:
	@echo "folder-together Derleme Komutları:"
	@echo "  make build          - Hem Linux hem Windows sürümlerini derler"
	@echo "  make build-linux    - Yalnızca Linux (Kubuntu) binary'sini derler"
	@echo "  make build-windows  - Yalnızca Windows 10 (.exe) binary'sini derler"
	@echo "  make run            - Geliştirme modunda doğrudan çalıştırır"
