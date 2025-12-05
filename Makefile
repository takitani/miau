.PHONY: run build test clean fmt vet lint

# Rodar sem compilar
run:
	go run ./cmd/miau

# Compilar binário
build:
	go build -o miau ./cmd/miau

# Rodar testes (unitários, rápidos)
test:
	go test ./... -short -v

# Rodar testes com coverage
test-coverage:
	go test ./... -short -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Rodar testes de um pacote específico
test-services:
	go test ./internal/services/... -v

# Rodar testes com race detector
test-race:
	go test ./... -short -race

# Rodar testes de integração (quando implementados)
test-integration:
	go test ./... -tags=integration -v

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

.PHONY: desktop-install desktop-dev desktop-build desktop-run desktop-build-windows desktop-build-all

# Install frontend dependencies
desktop-install:
	cd cmd/miau-desktop/frontend && npm install

# Run in development mode (hot reload)
# Uses webkit2_41 tag for Fedora 43+ compatibility (webkit2gtk-4.1)
# GODEBUG=asyncpreemptoff=1 prevents signal 11 crash with Go 1.24+/WebKit
desktop-dev:
	cd cmd/miau-desktop && GODEBUG=asyncpreemptoff=1 wails dev -tags webkit2_41

# Run in dev mode with devtools auto-open
desktop-dev-debug:
	cd cmd/miau-desktop && GODEBUG=asyncpreemptoff=1 wails dev -tags webkit2_41 -appargs "--devtools"

# Build desktop app for current platform
desktop-build:
	cd cmd/miau-desktop && wails build -tags webkit2_41

# Build desktop app with devtools enabled (F12 to open inspector)
desktop-build-debug:
	cd cmd/miau-desktop && wails build -tags webkit2_41 -devtools

# Run the built desktop app (with workaround for Go/WebKit signal conflict)
desktop-run:
	GODEBUG=asyncpreemptoff=1 cmd/miau-desktop/build/bin/miau-desktop

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
