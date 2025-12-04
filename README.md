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
 ↑↓:navegar  Tab:pastas  r:sync  a:AI  c:compor  q:sair
```

## Funcionalidades

### Core
- [x] Conexão IMAP com múltiplas contas
- [x] Download e armazenamento local de emails (SQLite)
- [x] Sincronização configurável (últimos X dias ou todos)
- [x] Busca full-text com FTS5 trigram (busca parcial)
- [x] Detecção de emails deletados no servidor
- [x] Arquivamento Gmail-style (e: arquivar, x: lixeira)
- [x] Retenção permanente de dados (nunca deleta nada)

### Envio de Emails
- [x] Envio via SMTP com autenticação
- [x] Envio via Gmail API (bypass DLP/classificação)
- [x] Assinaturas HTML e texto configuráveis
- [x] Classificação de emails (Google Workspace)
- [x] Detecção de bounce após envio

### TUI (Terminal User Interface)
- [x] Navegação por pastas/labels
- [x] Lista de emails com indicadores (lido/não lido/favorito)
- [x] Atalhos de teclado estilo vim (j/k)
- [x] Visualização de corpo do email (HTML no browser)
- [x] Composição de emails e respostas
- [x] Painel de AI integrado

### Autenticação
- [x] Login com senha/App Password
- [x] OAuth2 para Gmail/Google Workspace
- [x] Comando `miau auth` para gerenciar tokens

### Integração com IA (via Claude Code)
- [x] Chat integrado na TUI (tecla `a`)
- [x] Queries no banco de emails via linguagem natural
- [x] Criação de drafts via IA (responder emails)
- [x] Operações em lote com preview (arquivar/deletar múltiplos)
- [ ] Resumo de emails longos
- [ ] Categorização automática

## Stack Tecnológico

- **Linguagem**: Go
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) (Charm.sh)
- **Armazenamento**: SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite))
- **IMAP**: [go-imap/v2](https://github.com/emersion/go-imap)
- **SMTP**: net/smtp + PLAIN/LOGIN auth
- **Gmail API**: REST API para envio (bypass DLP)
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
# Executar TUI principal
miau

# Executar em modo debug
miau --debug

# Autenticação OAuth2 (para Gmail API)
miau auth

# Ver assinatura configurada
miau signature
```

### Atalhos de Teclado

| Tecla | Ação |
|-------|------|
| `j/k` ou `↑/↓` | Navegar na lista |
| `Enter` | Abrir email no browser |
| `Tab` | Alternar painel de pastas |
| `c` | Compor novo email |
| `r` | Sincronizar emails |
| `a` | Abrir painel de AI |
| `d` | Ver drafts pendentes |
| `e` | Arquivar email |
| `x` ou `#` | Mover para lixeira |
| `q` | Sair |

### Configuração

O arquivo de configuração fica em `~/.config/miau/config.yaml`:

```yaml
accounts:
  - name: minha-conta
    email: usuario@exemplo.com
    auth_type: oauth2  # ou "password"
    oauth2:
      client_id: "seu-client-id.apps.googleusercontent.com"
      client_secret: "seu-client-secret"
    send_method: gmail_api  # ou "smtp"
    imap:
      host: imap.gmail.com
      port: 993
      tls: true
    signature:
      enabled: true
      html: |
        <p>Atenciosamente,<br>Seu Nome</p>
      text: |
        Atenciosamente,
        Seu Nome
sync:
  interval: 5m
  initial_days: 30
ui:
  theme: dark
  page_size: 50
compose:
  format: html
```

## Gmail API vs SMTP

O miau suporta dois métodos de envio:

| Método | Vantagens | Desvantagens |
|--------|-----------|--------------|
| **SMTP** | Funciona com qualquer provedor | Pode ter problemas com DLP/classificação |
| **Gmail API** | Bypass de DLP, melhor integração | Requer OAuth2, só funciona com Google |

Para usar Gmail API, configure `send_method: gmail_api` e execute `miau auth` para autenticar.

## Licença

MIT

---

*Projeto criado para uso pessoal, gerenciado com Claude Code.*
