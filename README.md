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

## Funcionalidades Planejadas

### Core
- [ ] Conexão IMAP com múltiplas contas
- [ ] Download e armazenamento local de emails (SQLite/arquivos)
- [ ] Sincronização incremental
- [ ] Busca full-text local

### TUI (Terminal User Interface)
- [ ] Navegação por pastas/labels
- [ ] Visualização de emails
- [ ] Composição de respostas
- [ ] Atalhos de teclado estilo vim

### Integração com IA (via Claude Code)
- [ ] Resumo de emails longos
- [ ] Sugestão de respostas
- [ ] Categorização automática
- [ ] Extração de tarefas/ações
- [ ] Análise de threads de discussão

## Stack Tecnológico

- **Linguagem**: Go
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) (Charm.sh)
- **CLI**: [Cobra](https://github.com/spf13/cobra) para comandos
- **Armazenamento**: SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)) + arquivos .eml
- **IMAP**: [go-imap](https://github.com/emersion/go-imap)
- **Config**: [Viper](https://github.com/spf13/viper) para configuração

## Instalação

```bash
# Em breve
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
