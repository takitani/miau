# miau

**M**ail **I**ntelligence **A**ssistant **U**tility - Seu gerenciador de emails local com IA.

> "miau" - tem **IA** no meio, sacou? 🐱

## O que é?

**miau** é uma ferramenta CLI/TUI para baixar, armazenar e gerenciar seus emails localmente via IMAP, com integração ao Claude Code para ajudar a responder, organizar e analisar suas mensagens.

## Por que "miau"?

- É curto e fácil de digitar no terminal
- Tem "**IA**" escondido no meio (m-**ia**-u)
- É brasileiro e divertido
- Soa como um gato pedindo atenção... assim como seus emails não lidos

## Screenshots

```
┌─ miau 🐱  demo@exemplo.com  [INBOX] (15 emails) ─────────────────────────────┐
│ ★ miau Team          │ Bem-vindo ao miau! 🐱                    │ 03/12 14:30 │
│ ● Maria Silva        │ Re: Proposta comercial Q4 2025           │ 03/12 13:30 │
│ ● João Santos        │ Reunião amanhã às 14h confirmada         │ 03/12 12:30 │
│   Financeiro         │ Fatura #12345 - Dezembro/2025            │ 03/12 11:30 │
│   Tech Weekly        │ Newsletter: Novidades em IA              │ 03/12 10:30 │
│ ★ Segurança          │ Alerta: Login detectado em novo dispo... │ 03/12 09:30 │
│   Loja Online        │ Seu pedido foi enviado!                  │ 02/12 14:30 │
│ ● DevConf            │ Convite: Webinar sobre Go e TUI          │ 02/12 14:30 │
├─ AI ─────────────────────────────────────────────────────────────────────────┤
│ 🤖 AI: quantos emails não lidos?                                             │
│ > quantos emails não lidos?                                                  │
│                                                                              │
│ Você tem 5 emails não lidos na sua caixa de entrada.                         │
└──────────────────────────────────────────────────────────────────────────────┘
 ↑↓:navegar  Tab:pastas  r:sync  a:AI  q:sair
```

## Funcionalidades

### Core
- [x] Conexão IMAP com múltiplas contas
- [x] Download e armazenamento local de emails (SQLite)
- [x] Sincronização configurável (últimos X dias ou todos)
- [x] Busca full-text com FTS5 trigram (busca parcial)

### TUI (Terminal User Interface)
- [x] Navegação por pastas/labels
- [x] Lista de emails com indicadores (lido/não lido/favorito)
- [x] Atalhos de teclado estilo vim (j/k)
- [ ] Visualização de corpo do email
- [ ] Composição de respostas

### Integração com IA (via Claude Code)
- [x] Chat integrado na TUI (tecla `a`)
- [x] Queries no banco de emails via linguagem natural
- [ ] Resumo de emails longos
- [ ] Sugestão de respostas
- [ ] Categorização automática

## Stack Tecnológico

- **Linguagem**: Go
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) (Charm.sh)
- **CLI**: [Cobra](https://github.com/spf13/cobra) para comandos
- **Armazenamento**: SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)) + arquivos .eml
- **IMAP**: [go-imap](https://github.com/emersion/go-imap)
- **Config**: [Viper](https://github.com/spf13/viper) para configuração

## Dependências

- **Go** 1.21+
- **Claude Code** - CLI do Claude para integração com IA ([instalar](https://claude.ai/code))
- **sqlite3** - Driver do SQLite para queries via CLI

```bash
# Fedora/RHEL
sudo dnf install sqlite

# Ubuntu/Debian
sudo apt install sqlite3

# macOS
brew install sqlite3

# Windows (via winget)
winget install SQLite.SQLite

# Windows (via choco)
choco install sqlite
```

## Instalação

```bash
git clone https://github.com/takitani/miau.git
cd miau
go build ./cmd/miau/
./miau
```

## Uso

```bash
# Exemplos futuros
miau sync              # sincroniza emails
miau inbox             # abre TUI na inbox
miau search "projeto"  # busca local
miau reply 123         # responde email #123 com ajuda de IA
```

## Licença

MIT

---

*Projeto criado para uso pessoal, gerenciado com Claude Code.*
