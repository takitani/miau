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
