# Changelog

Histórico de implementação do miau, ordenado do mais recente para o mais antigo.

## [Unreleased]

### Adicionado
- **Auto-refresh com timer visual**: Sync automático a cada 60 segundos
  - Barra de progresso animada no footer (TUI e Desktop)
  - Indicador visual de novos emails após cada sync
  - Badge verde piscando para novos emails, cinza para "0 novos"
- **Sync logs**: Tabela de histórico de syncs para contagem precisa
  - `sync_logs` registra cada operação de sync
  - Contagem correta de emails novos baseada em `created_at`
  - Auto-limpeza de logs antigos (>7 dias)
- **Fix sync Desktop**: Botão "Sync" agora recarrega emails corretamente
  - Chama `loadEmails()` após sync completar
  - Sync inicial automático no startup
- **Desktop GUI (Wails + Svelte)**: Interface gráfica nativa para Windows/Linux/macOS
  - Layout 3 painéis: folders, lista de emails, visualizador
  - Carregamento de body sob demanda via IMAP (corrigido bug de mailbox selection)
  - Suporte a imagens inline (conversão cid: para data: URL)
  - Bloqueio de imagens externas por segurança (opt-in para mostrar)
  - Compose modal com reply/reply-all/forward
  - Atalhos de teclado: j/k navegar, r responder, c compor, e arquivar, x deletar
  - DevTools habilitado (F12) para debug
  - Workaround para Go 1.24+/WebKit signal conflict (GODEBUG=asyncpreemptoff=1)
  - Build: `make desktop-build-debug`, Run: `make desktop-run`
- **Arquitetura Modular (Ports/Adapters)**: Preparação para múltiplas interfaces
  - `internal/ports/`: Interfaces de domínio (EmailService, StoragePort, etc)
  - `internal/adapters/`: Implementações (IMAP, Storage)
  - `internal/services/`: Lógica de negócio (Sync, Send, Draft, Batch, Search)
  - `internal/app/`: Aplicação core que conecta tudo
  - Permite TUI, Web e Desktop compartilharem a mesma lógica
- **Image Preview no TUI**: Visualização de imagens diretamente no terminal
  - Tecla `i` no viewer abre preview de imagens
  - Renderização gráfica com chafa/viu (recomendado)
  - Fallback ASCII art nativo quando chafa não instalado
  - Dica de instalação exibida automaticamente
  - Navegação entre imagens com `←`/`→` ou `h`/`l`
  - Salvar imagem com `s` (salva em ~/Downloads)
  - Abrir no viewer externo com `Enter`
  - Suporte a JPEG, PNG, GIF, WebP
  - **Qualidade melhorada**: `--passthrough none`, `--work 9`, `--colors full`
- **ROADMAP.md**: Roadmap visual com barras de progresso e fila de prioridades
- **IDEAS.md**: Novas ideias adicionadas:
  - Multi-select (Space/Shift para selecionar múltiplos emails)
  - Suporte a mouse (click, scroll, context menu)
  - Help overlay (tecla ? com todos os atalhos)
  - About screen (info do autor, LinkedIn, Exato)
  - Undo/Redo infinito
  - Temas e customização visual
  - Multi-language (i18n)
  - Tasks/Todo integration
  - Calendar integration
  - Multi-AI integration (Gemini, Ollama, OpenRouter)
  - Scheduled Messages (Send Later)

### Corrigido
- **Delete/Archive não sincronizava com Gmail**: Agora usa Gmail API para OAuth2 (independente do SendMethod), com fallback para IMAP
- **Overlay de imagem**: Removido background que interferia com cores ANSI do chafa

### Em desenvolvimento
- Resumo automático de emails longos via IA
- Categorização automática de emails
- Busca fuzzy nativa (tecla F)

---

## 2024-12-04

### Arquivamento Gmail-style e Operações em Lote
**Commit:** `de0d314`

#### Adicionado
- **Arquivamento Gmail-style**: Separação semântica entre arquivar e deletar
  - `is_archived=1`: Remove do inbox, mantém no All Mail
  - `is_deleted=1`: Move para lixeira (30 dias antes de arquivar permanentemente)
  - Tecla `e` para arquivar, `x` ou `#` para deletar
- **Sync bidirecional**: Operações locais sincronizam com servidor
  - IMAP MOVE para arquivar/deletar
  - Gmail API para contas OAuth2
- **Retenção permanente de dados**: Nunca deletamos nada
  - `emails_archive`: Emails após purge do servidor
  - `drafts_history`: Histórico de todos os drafts
  - `sent_emails`: Registro de todos os emails enviados
  - Auto-purge: emails deletados há 30+ dias vão para archive
- **Operações em lote via IA**: Preview antes de executar
  - Tabela `pending_batch_ops` para operações pendentes
  - Banner no inbox mostra preview dos emails afetados
  - Tecla `y` confirma, `n` cancela
  - Suporta: archive, delete, mark_read, mark_unread
- **Drafts via IA**: Responder emails com linguagem natural
  - AI cria draft automaticamente ao responder
  - Tecla `d` mostra drafts pendentes
  - Editar, enviar ou cancelar drafts

#### Corrigido
- **NULL fields**: Campos nullable no Draft model corrigidos com `sql.NullString`
- **Race condition**: `emailsLoadedMsg` não sobrescreve quando filtro ativo
- **Closures em goroutines**: accountID capturado antes do closure

#### Alterado
- CLAUDE.md atualizado com instruções para escrita de emails (sem markdown)
- Footer do inbox mostra atalhos para archive/delete

---

## 2024-12-03

### Gmail API e Otimizações de Boot
**Commit:** `356cf65`

#### Adicionado
- **Gmail API Send**: Envio de emails via Gmail REST API como alternativa ao SMTP
  - Bypass de DLP/classificação do Google Workspace
  - Suporte a `classificationLabelValues` na API
  - Config `send_method: gmail_api` ou `smtp`
- **Comando `miau auth`**: Fluxo de autenticação OAuth2 fora da TUI
  - Gerenciamento de tokens OAuth2
  - Verificação de token existente antes de renovar
- **Detecção de emails deletados**: Sync detecta emails removidos no servidor
  - Usa `UIDSearch` para listar UIDs do servidor
  - Marca emails locais como `is_deleted=1` (soft delete)
  - Nunca faz hard delete no banco

#### Corrigido
- **Boot lento**: Otimizado de ~20s para <1s
  - Antes: buscava STATUS de todas as 80 pastas
  - Agora: só busca STATUS do INBOX
- **FetchNewEmails**: Corrigido para usar `UIDSearch` em vez de `Search`
- **GetAllUIDs**: Corrigido para usar `UIDSearch` para retornar UIDs corretamente
- **Bounce detection**: Melhorada validação por timestamp e destinatário

#### Alterado
- Scope OAuth2 agora inclui `gmail.send` além de `mail.google.com`
- Mensagem de sucesso diferencia Gmail API vs SMTP

---

## 2024-12-03

### Detecção de Bounce
**Commit:** `1de7fd0`

#### Adicionado
- **Bounce detection**: Monitora emails enviados por 5 minutos
  - Detecta mensagens de mailer-daemon, postmaster, etc
  - Overlay de alerta quando bounce é detectado
  - Log detalhado em `/tmp/miau-bounce.log`

---

## 2024-12-02

### Cliente SMTP e Composição
**Commit:** `0266c7c`

#### Adicionado
- **Cliente SMTP**: Envio de emails com autenticação PLAIN/LOGIN
- **Composição de emails**: Interface para escrever novos emails (tecla `c`)
- **Assinaturas**: Suporte a assinaturas HTML e texto no config
- **Classificação**: Headers de classificação para Google Workspace
- **Reply**: Responder emails mantendo threading (In-Reply-To, References)

---

## 2024-12-01

### Visualização de Emails
**Commit:** `a50db5a`

#### Adicionado
- **Spinner durante sync**: Feedback visual durante sincronização
- **Visualização HTML**: Abre email no browser padrão do sistema
- **Decodificação**: Suporte a quoted-printable, base64, charsets variados

---

## 2024-11-30

### Melhorias de UX
**Commit:** `7ccd8e6`

#### Adicionado
- Limpa input do AI após enviar pergunta
- Mostra última pergunta no histórico

---

## 2024-11-29

### Sync Configurável e Busca
**Commit:** `87d62ac`

#### Adicionado
- **Período de sync configurável**: `sync.interval` e `sync.initial_days`
- **FTS5 trigram**: Busca parcial de texto (ex: "proj" encontra "projeto")

---

## 2024-11-28

### Documentação
**Commits:** `4567ed0`, `bef0f6e`

#### Adicionado
- Instruções de instalação do sqlite3 para Windows
- Seção de dependências no README (Claude Code + sqlite3)

---

## 2024-11-27

### Assistente de IA
**Commit:** `817633e`

#### Adicionado
- **Painel de AI integrado**: Tecla `a` abre chat com Claude
- Queries em linguagem natural no banco de emails
- Acesso direto ao SQLite via Claude Code

---

## 2024-11-26

### Armazenamento Local
**Commit:** `f7ac66b`

#### Adicionado
- **SQLite storage**: Banco de dados local para emails
- Schema: accounts, folders, emails
- **FTS5**: Busca full-text nos emails
- Repository pattern para acesso a dados

---

## 2024-11-25

### Melhorias de Autenticação
**Commits:** `9114f40`, `6010285`

#### Adicionado
- URL de App Password clicável em terminais modernos
- Prompt de App Password em caso de falha de autenticação

---

## 2024-11-24

### IMAP e Interface
**Commit:** `45db4f1`

#### Adicionado
- **Cliente IMAP**: Conexão e listagem de emails
- **TUI Inbox**: Interface principal com lista de emails
- Navegação por pastas/labels
- Indicadores de lido/não lido/favorito

---

## 2024-11-23

### OAuth2
**Commit:** `8288f3c`

#### Adicionado
- **OAuth2**: Suporte a autenticação OAuth2 para Gmail/Google Workspace
- Fluxo de autorização via browser
- Armazenamento seguro de tokens
- XOAUTH2 SASL para IMAP

---

## 2024-11-22

### Setup Wizard
**Commit:** `3827828`

#### Adicionado
- **Setup wizard**: Configuração inicial guiada
- Auto-detecção de servidores IMAP
- Suporte a múltiplos provedores

---

## 2024-11-21

### Projeto Inicial
**Commits:** `a041592`, `77f3920`

#### Adicionado
- Estrutura inicial do projeto
- Makefile para comandos de desenvolvimento
- Configuração básica com Viper

---

## Roadmap

### Próximas Features
- [x] ~~Arquivar emails (mover para All Mail)~~ ✅
- [x] ~~Mover para lixeira~~ ✅
- [ ] Marcar como lido/não lido via interface
- [ ] Favoritar/desfavoritar
- [ ] Busca fuzzy nativa (tecla F)
- [ ] Múltiplas contas na interface
- [ ] Notificações de novos emails
- [ ] Modo offline completo
- [ ] Exportação de emails
- [ ] Filtros e regras automáticas

### Melhorias Técnicas
- [ ] Testes unitários
- [ ] Testes de integração
- [ ] CI/CD pipeline
- [ ] Releases automatizados
- [ ] Documentação de API interna
