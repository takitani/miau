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
# Desktop App (Wails v3 + Svelte)
# ============================================================================

.PHONY: desktop-install desktop-dev desktop-build desktop-run desktop-build-windows desktop-build-all

# Install frontend dependencies
desktop-install:
	cd cmd/miau-desktop/frontend && npm install

# Run in development mode (hot reload)
# Wails v3 uses Taskfile.yml for configuration
desktop-dev:
	cd cmd/miau-desktop && wails3 dev

# Run in dev mode with devtools auto-open
desktop-dev-debug:
	cd cmd/miau-desktop && wails3 dev -- --devtools

# Build desktop app for current platform
desktop-build:
	cd cmd/miau-desktop && wails3 build

# Run the built desktop app
desktop-run:
	cmd/miau-desktop/bin/miau-desktop

# Run with devtools
desktop-run-debug:
	cmd/miau-desktop/bin/miau-desktop --devtools

# Build for Windows (cross-compile requires Docker setup)
desktop-build-windows:
	cd cmd/miau-desktop && wails3 task windows:build

# Build for Linux
desktop-build-linux:
	cd cmd/miau-desktop && wails3 task linux:build

# Build for macOS
desktop-build-darwin:
	cd cmd/miau-desktop && wails3 task darwin:build

# Package for distribution (creates installer/package)
desktop-package:
	cd cmd/miau-desktop && wails3 task package
