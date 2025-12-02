# miau - Instruções para Claude Code

## Sobre o Projeto

**miau** (Mail Intelligence Assistant Utility) é um cliente de email local via IMAP com interface TUI e integração com IA para auxiliar na gestão de emails.

## Convenções do Projeto

### Linguagem
- Código e comentários em inglês
- Documentação para usuário em português (pt-BR)
- Mensagens de commit em inglês

### Estilo de Código
- Seguir convenções Go (gofmt, go vet, golint)
- Preferir simplicidade sobre abstração prematura
- Código autoexplicativo, comentários apenas quando necessário
- Usar `var` para declarações quando possível
- Tratamento de erros explícito (não ignorar erros)

### Estrutura de Pastas
```
miau/
├── cmd/
│   └── miau/
│       └── main.go       # entrypoint
├── internal/
│   ├── config/           # configuração (viper)
│   ├── imap/             # cliente IMAP
│   ├── storage/          # SQLite + .eml
│   ├── tui/              # Bubble Tea models/views
│   │   ├── inbox/
│   │   ├── reader/
│   │   └── compose/
│   └── ai/               # integração com Claude
├── pkg/                  # código reutilizável público
├── docs/                 # documentação
├── config.example.yaml   # exemplo de configuração
├── go.mod
├── go.sum
└── data/                 # diretório local de emails (gitignore)
```

### Segurança - CRÍTICO
- **NUNCA** commitar credenciais, senhas ou tokens
- Senhas IMAP devem usar variáveis de ambiente ou keyring do sistema
- Arquivos de configuração com credenciais devem estar no .gitignore
- Emails baixados são dados sensíveis - não commitar

### Funcionalidades Prioritárias
1. Conexão IMAP básica e download de emails
2. Armazenamento local estruturado
3. TUI para navegação
4. Integração com Claude para respostas

### Comandos Úteis
```bash
# Desenvolvimento
go run ./cmd/miau              # executar
go build -o miau ./cmd/miau    # compilar
go test ./...                  # testes
go mod tidy                    # limpar dependências

# Formatação e lint
gofmt -w .
go vet ./...
```

### Dependências Principais
```go
// TUI
github.com/charmbracelet/bubbletea
github.com/charmbracelet/lipgloss
github.com/charmbracelet/bubbles

// CLI
github.com/spf13/cobra
github.com/spf13/viper

// IMAP
github.com/emersion/go-imap/v2
github.com/emersion/go-message

// Storage
modernc.org/sqlite
```

## Contexto para IA

Quando ajudando com este projeto:
- O objetivo é ter controle LOCAL dos emails, não depender de webmail
- A integração com IA é para AUXILIAR, não automatizar completamente
- Privacidade é importante - tudo roda local
- O usuário quer poder usar Claude Code para ajudar a responder emails complexos
