.PHONY: run build test clean fmt vet lint

# Rodar sem compilar
run:
	go run ./cmd/miau

# Compilar binário
build:
	go build -o miau ./cmd/miau

# Rodar testes
test:
	go test ./...

# Limpar binários
clean:
	rm -f miau

# Formatar código
fmt:
	gofmt -w .

# Verificar problemas
vet:
	go vet ./...

# Formatar + verificar
lint: fmt vet

# Atualizar dependências
tidy:
	go mod tidy

# Build para Windows (cross-compile)
build-windows:
	GOOS=windows GOARCH=amd64 go build -o miau.exe ./cmd/miau

# ============================================================================
# Desktop App (Wails + Svelte)
# ============================================================================

.PHONY: desktop-install desktop-dev desktop-build desktop-build-windows desktop-build-all

# Install frontend dependencies
desktop-install:
	cd cmd/miau-desktop/frontend && npm install

# Run in development mode (hot reload)
# Uses webkit2_41 tag for Fedora 43+ compatibility (webkit2gtk-4.1)
desktop-dev:
	cd cmd/miau-desktop && wails dev -tags webkit2_41

# Build desktop app for current platform
desktop-build:
	cd cmd/miau-desktop && wails build -tags webkit2_41

# Build for Windows (no webkit tag needed)
desktop-build-windows:
	cd cmd/miau-desktop && wails build -platform windows/amd64

# Build for Linux
desktop-build-linux:
	cd cmd/miau-desktop && wails build -platform linux/amd64 -tags webkit2_41

# Build for all platforms
desktop-build-all:
	cd cmd/miau-desktop && wails build -platform linux/amd64 -tags webkit2_41
	cd cmd/miau-desktop && wails build -platform windows/amd64
	cd cmd/miau-desktop && wails build -platform darwin/universal
