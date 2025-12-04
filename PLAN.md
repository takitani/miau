# PLANO: miau Desktop App (Wails + Svelte)

## Visão Geral

Criar uma interface desktop para o miau usando **Wails + Svelte**, aproveitando 100% da arquitetura modular existente (ports/adapters). O frontend Svelte poderá ser reaproveitado para web no futuro.

## Por que Wails + Svelte?

| Critério | Decisão |
|----------|---------|
| **Web futura** | Wails - frontend reaproveitável |
| **Binário** | ~4-8MB (vs ~100MB Electron) |
| **Windows** | WebView2 (Edge) - excelente suporte |
| **Performance** | Svelte compila para JS vanilla, zero overhead |
| **Aprendizado** | Svelte é o framework mais simples |
| **Teclado-first** | Total controle sobre shortcuts |

## Arquitetura

```
┌─────────────────────────────────────────────────────────────────┐
│                        Wails Desktop App                         │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Frontend (Svelte)                         ││
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐    ││
│  │  │  Inbox    │ │  Viewer   │ │  Compose  │ │  Search   │    ││
│  │  │  List     │ │  Panel    │ │  Modal    │ │  Panel    │    ││
│  │  └───────────┘ └───────────┘ └───────────┘ └───────────┘    ││
│  │                         │                                    ││
│  │              Wails Bindings (TypeScript)                     ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Backend (Go)                              ││
│  │                                                              ││
│  │    ┌─────────────────────────────────────────────┐          ││
│  │    │           internal/app.Application          │          ││
│  │    │     (100% reutilizado, zero alterações)     │          ││
│  │    └─────────────────────────────────────────────┘          ││
│  │                          │                                   ││
│  │    ┌──────────┬──────────┬──────────┬──────────┐            ││
│  │    │ Email    │ Send     │ Search   │ Sync     │            ││
│  │    │ Service  │ Service  │ Service  │ Service  │            ││
│  │    └──────────┴──────────┴──────────┴──────────┘            ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Estrutura de Diretórios

```
miau/
├── cmd/
│   ├── miau/              # TUI existente (não modificar)
│   └── miau-desktop/      # NOVO: Entry point Wails
│       └── main.go
│
├── internal/
│   ├── app/               # Reutilizar 100%
│   ├── ports/             # Reutilizar 100%
│   ├── services/          # Reutilizar 100%
│   ├── adapters/          # Reutilizar 100%
│   ├── storage/           # Reutilizar 100%
│   ├── imap/              # Reutilizar 100%
│   ├── smtp/              # Reutilizar 100%
│   ├── gmail/             # Reutilizar 100%
│   ├── auth/              # Reutilizar 100%
│   ├── config/            # Reutilizar 100%
│   ├── tui/               # Não modificar
│   │
│   └── desktop/           # NOVO: Backend Wails
│       ├── app.go         # Wails app struct
│       ├── bindings.go    # Métodos expostos ao frontend
│       ├── events.go      # Event handling (Go → JS)
│       └── types.go       # DTOs para frontend
│
├── desktop/               # NOVO: Frontend Svelte
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.js
│   ├── src/
│   │   ├── App.svelte     # Root component
│   │   ├── main.js        # Entry point
│   │   ├── lib/
│   │   │   ├── stores/    # Svelte stores (state)
│   │   │   │   ├── emails.js
│   │   │   │   ├── folders.js
│   │   │   │   └── ui.js
│   │   │   ├── components/
│   │   │   │   ├── EmailList.svelte
│   │   │   │   ├── EmailRow.svelte
│   │   │   │   ├── EmailViewer.svelte
│   │   │   │   ├── FolderList.svelte
│   │   │   │   ├── SearchPanel.svelte
│   │   │   │   ├── ComposeModal.svelte
│   │   │   │   ├── Toolbar.svelte
│   │   │   │   └── StatusBar.svelte
│   │   │   └── utils/
│   │   │       ├── keyboard.js
│   │   │       └── wailsEvents.js
│   │   └── styles/
│   │       ├── global.css
│   │       └── theme.css
│   └── wailsjs/           # Auto-gerado pelo Wails
│       └── go/
│           └── desktop/
│               └── App.d.ts
│
├── wails.json             # NOVO: Config Wails
├── Makefile               # Atualizar com comandos desktop
└── build/                 # Binários desktop
```

## Fases de Implementação

### Fase 1: Setup Wails + Estrutura Base
**Objetivo:** Projeto Wails funcionando com Svelte vazio

1. Instalar Wails CLI
2. Inicializar projeto Wails com Svelte
3. Criar `cmd/miau-desktop/main.go`
4. Criar `internal/desktop/app.go` básico
5. Testar build Windows/Linux

**Entregável:** `miau-desktop` abre janela vazia

### Fase 2: Bindings Go ↔ Svelte
**Objetivo:** Conectar backend Go ao frontend

1. Criar `internal/desktop/bindings.go` com métodos:
   - `GetFolders() []Folder`
   - `GetEmails(folder string, limit int) []Email`
   - `GetEmail(id int64) Email`
   - `MarkAsRead(id int64, read bool)`
   - `Archive(id int64)`
   - `Delete(id int64)`
   - `Search(query string, limit int) []Email`
   - `Connect()`
   - `Sync(folder string)`
   - `SendEmail(to, subject, body string)`

2. Configurar Wails events (Go → JS):
   - `email:new`
   - `email:updated`
   - `sync:started`
   - `sync:completed`
   - `connection:status`

**Entregável:** Console JS consegue chamar funções Go

### Fase 3: UI Base - Lista de Emails
**Objetivo:** Inbox funcional com navegação por teclado

1. Layout principal (3 painéis):
   - Esquerda: Folders (colapsável)
   - Centro: Lista de emails
   - Direita: Preview (colapsável)

2. Componentes:
   - `EmailList.svelte` com virtual scroll
   - `EmailRow.svelte` (from, subject, date, flags)
   - `FolderList.svelte`

3. Keyboard navigation:
   - `j/k` ou `↑/↓`: navegar lista
   - `Enter`: abrir email
   - `Tab`: alternar painéis
   - `/`: abrir busca
   - `Esc`: fechar modais

**Entregável:** Navegar emails com teclado

### Fase 4: Visualização de Email
**Objetivo:** Ver conteúdo do email selecionado

1. `EmailViewer.svelte`:
   - Headers (From, To, CC, Date, Subject)
   - Body HTML renderizado (sanitizado)
   - Body text fallback
   - Attachments list

2. Ações inline:
   - `r`: reply
   - `R`: reply all
   - `f`: forward
   - `e`: archive
   - `x/#`: delete
   - `s`: star

**Entregável:** Visualizar email completo

### Fase 5: Busca Fuzzy (ninja mode)
**Objetivo:** Busca rápida como no TUI

1. `SearchPanel.svelte`:
   - Input com debounce (150ms)
   - Resultados em tempo real
   - Highlight de matches
   - Navegação por teclado nos resultados

2. Atalhos:
   - `/`: abre busca
   - `Enter`: seleciona resultado
   - `↑/↓`: navega resultados
   - `Esc`: fecha busca

3. Backend:
   - Usar `SearchService.Search()` existente
   - FTS5 trigram já implementado

**Entregável:** Busca instantânea como TUI

### Fase 6: Composição de Email
**Objetivo:** Enviar emails

1. `ComposeModal.svelte`:
   - To, CC, BCC (autocomplete de contatos)
   - Subject
   - Body (textarea ou rich editor simples)
   - Assinatura automática

2. Ações:
   - `c`: novo email
   - `r`: reply (preenche To, Subject)
   - `R`: reply all
   - `f`: forward
   - `Ctrl+Enter`: enviar
   - `Ctrl+Shift+D`: salvar draft

**Entregável:** Enviar emails via Gmail API ou SMTP

### Fase 7: Sync e Status
**Objetivo:** Sincronização visual

1. `StatusBar.svelte`:
   - Status de conexão
   - Último sync
   - Progresso de sync
   - Notificações

2. Background sync:
   - Auto-sync a cada 5 min
   - Manual com `r`
   - Progress indicator

**Entregável:** Sync funcionando

### Fase 8: Polish e Extras
**Objetivo:** Experiência final

1. Temas (dark/light)
2. Settings modal
3. Multi-account selector
4. Drag & drop (mover para folder)
5. Context menu (right-click)
6. Notificações desktop (new email)
7. Tray icon (minimize to tray)

**Entregável:** App completo

## Keyboard Shortcuts (Spec)

| Key | Global | Inbox | Viewer | Search | Compose |
|-----|--------|-------|--------|--------|---------|
| `j` / `↓` | - | Next email | Scroll down | Next result | - |
| `k` / `↑` | - | Prev email | Scroll up | Prev result | - |
| `Enter` | - | Open email | - | Select | - |
| `/` | Open search | Open search | Open search | - | - |
| `Esc` | Close modal | - | Close | Close | Close |
| `c` | Compose | Compose | Compose | - | - |
| `r` | - | Reply | Reply | - | - |
| `R` | - | Reply all | Reply all | - | - |
| `f` | - | Forward | Forward | - | - |
| `e` | - | Archive | Archive | - | - |
| `x` / `#` | - | Delete | Delete | - | - |
| `s` | - | Star | Star | - | - |
| `u` | - | Mark unread | Mark unread | - | - |
| `Tab` | Switch panel | Switch panel | - | - | Next field |
| `g i` | Go to Inbox | - | - | - | - |
| `g s` | Go to Sent | - | - | - | - |
| `g d` | Go to Drafts | - | - | - | - |
| `S` | Settings | - | - | - | - |
| `?` | Help | Help | Help | - | - |
| `Ctrl+Enter` | - | - | - | - | Send |
| `q` | Quit | - | Close | - | - |

## Wails Bindings (Go)

```go
// internal/desktop/bindings.go

type DesktopApp struct {
    app ports.App
    ctx context.Context
}

// Folders
func (a *DesktopApp) GetFolders() ([]FolderDTO, error)
func (a *DesktopApp) SelectFolder(name string) error

// Emails
func (a *DesktopApp) GetEmails(folder string, limit int) ([]EmailDTO, error)
func (a *DesktopApp) GetEmail(id int64) (*EmailDetailDTO, error)
func (a *DesktopApp) GetEmailByUID(uid uint32) (*EmailDetailDTO, error)

// Actions
func (a *DesktopApp) MarkAsRead(id int64, read bool) error
func (a *DesktopApp) MarkAsStarred(id int64, starred bool) error
func (a *DesktopApp) Archive(id int64) error
func (a *DesktopApp) Delete(id int64) error
func (a *DesktopApp) MoveToFolder(id int64, folder string) error

// Search
func (a *DesktopApp) Search(query string, limit int) ([]EmailDTO, error)

// Sync
func (a *DesktopApp) Connect() error
func (a *DesktopApp) Disconnect() error
func (a *DesktopApp) SyncFolder(folder string) error
func (a *DesktopApp) GetConnectionStatus() ConnectionStatus

// Send
func (a *DesktopApp) SendEmail(req SendRequest) (*SendResult, error)
func (a *DesktopApp) SaveDraft(draft DraftDTO) (int64, error)
func (a *DesktopApp) SendDraft(draftID int64) (*SendResult, error)

// Config
func (a *DesktopApp) GetAccounts() []AccountDTO
func (a *DesktopApp) GetCurrentAccount() *AccountDTO
```

## DTOs (Transfer Objects)

```go
// internal/desktop/types.go

type EmailDTO struct {
    ID           int64     `json:"id"`
    UID          uint32    `json:"uid"`
    Subject      string    `json:"subject"`
    FromName     string    `json:"fromName"`
    FromEmail    string    `json:"fromEmail"`
    Date         time.Time `json:"date"`
    IsRead       bool      `json:"isRead"`
    IsStarred    bool      `json:"isStarred"`
    HasAttach    bool      `json:"hasAttachments"`
    Snippet      string    `json:"snippet"`
}

type EmailDetailDTO struct {
    EmailDTO
    ToAddresses  string    `json:"toAddresses"`
    CcAddresses  string    `json:"ccAddresses"`
    BodyText     string    `json:"bodyText"`
    BodyHTML     string    `json:"bodyHtml"`
    Attachments  []AttachmentDTO `json:"attachments"`
}

type FolderDTO struct {
    ID           int64  `json:"id"`
    Name         string `json:"name"`
    TotalMsgs    int    `json:"totalMessages"`
    UnreadMsgs   int    `json:"unreadMessages"`
}

type SendRequest struct {
    To      []string `json:"to"`
    Cc      []string `json:"cc"`
    Bcc     []string `json:"bcc"`
    Subject string   `json:"subject"`
    Body    string   `json:"body"`
    IsHTML  bool     `json:"isHtml"`
    ReplyTo int64    `json:"replyTo,omitempty"`
}
```

## Events (Go → JavaScript)

```go
// internal/desktop/events.go

// Emitir eventos para o frontend
func (a *DesktopApp) setupEventForwarding() {
    // Subscribe to app events and forward to Wails
    a.app.Events().SubscribeAll(func(evt ports.Event) {
        switch e := evt.(type) {
        case *ports.NewEmailEvent:
            runtime.EventsEmit(a.ctx, "email:new", e.Email)
        case *ports.EmailReadEvent:
            runtime.EventsEmit(a.ctx, "email:read", e.EmailID, e.Read)
        case *ports.SyncCompletedEvent:
            runtime.EventsEmit(a.ctx, "sync:completed", e.Folder, e.NewCount)
        case *ports.ConnectedEvent:
            runtime.EventsEmit(a.ctx, "connection:connected")
        case *ports.DisconnectedEvent:
            runtime.EventsEmit(a.ctx, "connection:disconnected")
        // ... more events
        }
    })
}
```

## Svelte Stores

```javascript
// desktop/src/lib/stores/emails.js
import { writable, derived } from 'svelte/store';
import { GetEmails, GetEmail } from '../../wailsjs/go/desktop/DesktopApp';

export const emails = writable([]);
export const selectedEmailId = writable(null);
export const currentFolder = writable('INBOX');

export const selectedEmail = derived(
    [emails, selectedEmailId],
    ([$emails, $id]) => $emails.find(e => e.id === $id)
);

export async function loadEmails(folder, limit = 50) {
    const result = await GetEmails(folder, limit);
    emails.set(result);
}

// Listen for events from Go
import { EventsOn } from '../../wailsjs/runtime/runtime';

EventsOn('email:new', (email) => {
    emails.update(list => [email, ...list]);
});
```

## Makefile Updates

```makefile
# Desktop commands
.PHONY: desktop-dev desktop-build desktop-build-windows

desktop-dev:
	cd desktop && wails dev

desktop-build:
	cd desktop && wails build

desktop-build-windows:
	cd desktop && wails build -platform windows/amd64

desktop-build-all:
	cd desktop && wails build -platform linux/amd64
	cd desktop && wails build -platform windows/amd64
	cd desktop && wails build -platform darwin/universal
```

## Timeline Estimada

| Fase | Descrição | Complexidade |
|------|-----------|--------------|
| 1 | Setup Wails + Estrutura | Média |
| 2 | Bindings Go ↔ Svelte | Média |
| 3 | UI Base - Lista | Alta |
| 4 | Visualização Email | Média |
| 5 | Busca Fuzzy | Média |
| 6 | Composição | Alta |
| 7 | Sync e Status | Baixa |
| 8 | Polish | Média |

## Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| WebView2 no Windows | Wails inclui installer automático |
| Performance virtual scroll | Usar svelte-virtual-list ou similar |
| Renderização HTML segura | DOMPurify para sanitizar |
| OAuth2 no desktop | Reutilizar auth existente (browser flow) |

## Decisões Técnicas

1. **Virtual Scroll**: Usar para listas grandes (>1000 emails)
2. **HTML Sanitization**: DOMPurify para body HTML
3. **State Management**: Svelte stores (não precisa Redux)
4. **CSS**: Vanilla CSS ou Tailwind (decidir na fase 3)
5. **Icons**: Heroicons ou similar (SVG)
6. **Rich Text Editor**: Tiptap ou textarea simples (fase 6)

## Próximos Passos

1. Aprovar este plano
2. Instalar Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
3. Iniciar Fase 1

---

*Plano criado em: 2025-12-04*
