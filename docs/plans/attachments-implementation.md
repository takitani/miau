# Plano de Implementação de Suporte a Anexos (Attachments) - miau

> **Data:** 2024-12-04
> **Status:** Planejado
> **Prioridade:** Alta

## Sumário Executivo

O miau atualmente:
- **Já possui parsing de anexos** no `internal/email/parser.go` - extrai anexos de emails MIME
- **Já possui estruturas de dados** para anexos em `ports.Attachment` e `desktop.AttachmentDTO`
- **Desktop já exibe anexos** inline (para imagens) e lista outros anexos
- **TUI já suporta preview de imagens** com chafa/viu
- **Não persiste anexos no banco** - sempre busca sob demanda via IMAP

O objetivo é adicionar persistência de metadados de anexos no SQLite para:
1. Mostrar indicador `has_attachments` de forma confiável
2. Permitir listar anexos sem precisar baixar email completo
3. Download sob demanda do conteúdo binário
4. Cache local opcional para anexos frequentemente acessados

---

## 1. Schema SQL para Tabela de Attachments

```sql
-- Tabela de metadados de anexos
CREATE TABLE IF NOT EXISTS attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email_id INTEGER NOT NULL,
    account_id INTEGER NOT NULL,

    -- Identificação
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    content_id TEXT,              -- Para imagens inline (cid:xxx)
    content_disposition TEXT,     -- 'attachment' ou 'inline'
    part_number TEXT,             -- "1.2" formato IMAP para download direto

    -- Metadados
    size INTEGER NOT NULL DEFAULT 0,
    checksum TEXT,                -- SHA256 do conteúdo (para deduplicação)
    encoding TEXT,                -- 'base64', 'quoted-printable', '7bit', '8bit'
    charset TEXT,                 -- Para anexos text/*

    -- Flags
    is_inline BOOLEAN DEFAULT 0,
    is_downloaded BOOLEAN DEFAULT 0,
    is_cached BOOLEAN DEFAULT 0,

    -- Cache local
    cache_path TEXT,              -- Caminho no disco se cached
    cached_at DATETIME,

    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    UNIQUE(email_id, filename)
);

CREATE INDEX IF NOT EXISTS idx_attachments_email ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_account ON attachments(account_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_type ON attachments(content_type);
CREATE INDEX IF NOT EXISTS idx_attachments_checksum ON attachments(checksum);
CREATE INDEX IF NOT EXISTS idx_attachments_inline ON attachments(is_inline);

-- Tabela de cache de conteúdo binário (opcional, para anexos pequenos)
-- Usar para imagens inline que são exibidas frequentemente
CREATE TABLE IF NOT EXISTS attachment_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    attachment_id INTEGER NOT NULL UNIQUE,
    data BLOB NOT NULL,
    compressed BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_accessed DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_attachment_cache_last_accessed ON attachment_cache(last_accessed);
```

---

## 2. Modificações no IMAP Client

### 2.1 Arquivo: `internal/imap/client.go`

Adicionar método para buscar estrutura BODYSTRUCTURE sem baixar conteúdo:

```go
// AttachmentInfo contains metadata about an attachment
type AttachmentInfo struct {
    PartNumber  string // "1.2" formato IMAP
    Filename    string
    ContentType string
    ContentID   string
    Encoding    string
    Size        int64
    IsInline    bool
    Charset     string
}

// FetchAttachmentMetadata busca metadados de anexos sem baixar conteúdo
func (c *Client) FetchAttachmentMetadata(uid uint32) ([]AttachmentInfo, error) {
    var uidSet = imap.UIDSet{}
    uidSet.AddNum(imap.UID(uid))

    var fetchOptions = &imap.FetchOptions{
        BodyStructure: &imap.FetchItemBodyStructure{Extended: true},
    }

    var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
    var messages, err = fetchCmd.Collect()
    if err != nil {
        return nil, err
    }

    if len(messages) == 0 {
        return nil, fmt.Errorf("email not found")
    }

    var msg = messages[0]
    return parseBodyStructure(msg.BodyStructure), nil
}

// FetchAttachmentPart baixa parte específica de anexo
func (c *Client) FetchAttachmentPart(uid uint32, partNumber string) ([]byte, error) {
    var uidSet = imap.UIDSet{}
    uidSet.AddNum(imap.UID(uid))

    var bodySection = &imap.FetchItemBodySection{
        Part: parsePartNumber(partNumber),
    }

    var fetchOptions = &imap.FetchOptions{
        BodySection: []*imap.FetchItemBodySection{bodySection},
    }

    var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
    var messages, err = fetchCmd.Collect()
    if err != nil {
        return nil, err
    }

    if len(messages) == 0 {
        return nil, fmt.Errorf("email not found")
    }

    var msg = messages[0]
    return msg.FindBodySection(bodySection), nil
}
```

### 2.2 Helper para parsear BODYSTRUCTURE

```go
// parseBodyStructure extrai informações de anexos da estrutura MIME
func parseBodyStructure(bs *imap.BodyStructure) []AttachmentInfo {
    var attachments []AttachmentInfo
    parseBodyStructureRecursive(bs, "", &attachments)
    return attachments
}

func parseBodyStructureRecursive(bs *imap.BodyStructure, prefix string, attachments *[]AttachmentInfo) {
    if bs == nil {
        return
    }

    // Se é multipart, processa cada parte
    if len(bs.Children) > 0 {
        for i, child := range bs.Children {
            var partNum string
            if prefix == "" {
                partNum = fmt.Sprintf("%d", i+1)
            } else {
                partNum = fmt.Sprintf("%s.%d", prefix, i+1)
            }
            parseBodyStructureRecursive(child, partNum, attachments)
        }
        return
    }

    // Verifica se é anexo
    var isAttachment = false
    var isInline = false
    var filename string

    // Content-Disposition
    if bs.Disposition != "" {
        var disp = strings.ToLower(bs.Disposition)
        isAttachment = disp == "attachment"
        isInline = disp == "inline"
        if params := bs.DispositionParams; params != nil {
            filename = params["filename"]
        }
    }

    // Fallback: filename em Content-Type params
    if filename == "" && bs.Params != nil {
        filename = bs.Params["name"]
    }

    // ContentID indica inline
    if bs.ID != "" {
        isInline = true
    }

    // Considera anexo se tem filename ou é imagem/audio/video
    var mediaType = strings.ToLower(bs.Type + "/" + bs.SubType)
    var isMedia = strings.HasPrefix(mediaType, "image/") ||
                  strings.HasPrefix(mediaType, "audio/") ||
                  strings.HasPrefix(mediaType, "video/") ||
                  strings.HasPrefix(mediaType, "application/")

    if isAttachment || (filename != "" || isMedia && (isInline || bs.Size > 0)) {
        *attachments = append(*attachments, AttachmentInfo{
            PartNumber:  prefix,
            Filename:    filename,
            ContentType: mediaType,
            ContentID:   strings.Trim(bs.ID, "<>"),
            Encoding:    strings.ToLower(bs.Encoding),
            Size:        int64(bs.Size),
            IsInline:    isInline,
            Charset:     bs.Params["charset"],
        })
    }
}
```

---

## 3. Modificações no Storage

### 3.1 Models: `internal/storage/models.go`

```go
// Attachment representa um anexo de email
type Attachment struct {
    ID                 int64          `db:"id"`
    EmailID            int64          `db:"email_id"`
    AccountID          int64          `db:"account_id"`
    Filename           string         `db:"filename"`
    ContentType        string         `db:"content_type"`
    ContentID          sql.NullString `db:"content_id"`
    ContentDisposition sql.NullString `db:"content_disposition"`
    PartNumber         sql.NullString `db:"part_number"`
    Size               int64          `db:"size"`
    Checksum           sql.NullString `db:"checksum"`
    Encoding           sql.NullString `db:"encoding"`
    Charset            sql.NullString `db:"charset"`
    IsInline           bool           `db:"is_inline"`
    IsDownloaded       bool           `db:"is_downloaded"`
    IsCached           bool           `db:"is_cached"`
    CachePath          sql.NullString `db:"cache_path"`
    CachedAt           sql.NullTime   `db:"cached_at"`
    CreatedAt          SQLiteTime     `db:"created_at"`
}

// AttachmentSummary versão resumida para listagem
type AttachmentSummary struct {
    ID          int64  `db:"id"`
    Filename    string `db:"filename"`
    ContentType string `db:"content_type"`
    Size        int64  `db:"size"`
    IsInline    bool   `db:"is_inline"`
    IsCached    bool   `db:"is_cached"`
}
```

### 3.2 Repository: `internal/storage/repository.go`

```go
// === ATTACHMENTS ===

// UpsertAttachment cria ou atualiza metadados de anexo
func UpsertAttachment(att *Attachment) error {
    _, err := db.Exec(`
        INSERT INTO attachments (
            email_id, account_id, filename, content_type, content_id,
            content_disposition, part_number, size, checksum, encoding, charset,
            is_inline, is_downloaded, is_cached
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(email_id, filename) DO UPDATE SET
            content_type = excluded.content_type,
            size = excluded.size,
            checksum = excluded.checksum`,
        att.EmailID, att.AccountID, att.Filename, att.ContentType, att.ContentID,
        att.ContentDisposition, att.PartNumber, att.Size, att.Checksum, att.Encoding, att.Charset,
        att.IsInline, att.IsDownloaded, att.IsCached)
    return err
}

// GetAttachmentsByEmail retorna anexos de um email
func GetAttachmentsByEmail(emailID int64) ([]AttachmentSummary, error) {
    var attachments []AttachmentSummary
    err := db.Select(&attachments, `
        SELECT id, filename, content_type, size, is_inline, is_cached
        FROM attachments
        WHERE email_id = ?
        ORDER BY is_inline DESC, filename`, emailID)
    return attachments, err
}

// GetAttachmentByID retorna anexo completo por ID
func GetAttachmentByID(id int64) (*Attachment, error) {
    var att Attachment
    err := db.Get(&att, "SELECT * FROM attachments WHERE id = ?", id)
    if err != nil {
        return nil, err
    }
    return &att, nil
}

// CountAttachmentsByEmail retorna quantidade de anexos
func CountAttachmentsByEmail(emailID int64) (int, error) {
    var count int
    err := db.Get(&count, "SELECT COUNT(*) FROM attachments WHERE email_id = ?", emailID)
    return count, err
}

// MarkAttachmentDownloaded marca anexo como baixado
func MarkAttachmentDownloaded(id int64) error {
    _, err := db.Exec(`UPDATE attachments SET is_downloaded = 1 WHERE id = ?`, id)
    return err
}

// CacheAttachmentContent salva conteúdo no cache
func CacheAttachmentContent(attachmentID int64, data []byte, compressed bool) error {
    _, err := db.Exec(`
        INSERT INTO attachment_cache (attachment_id, data, compressed)
        VALUES (?, ?, ?)
        ON CONFLICT(attachment_id) DO UPDATE SET
            data = excluded.data,
            compressed = excluded.compressed,
            last_accessed = CURRENT_TIMESTAMP`,
        attachmentID, data, compressed)
    if err != nil {
        return err
    }

    // Atualiza flag no attachment
    _, err = db.Exec(`
        UPDATE attachments SET is_cached = 1, cached_at = CURRENT_TIMESTAMP WHERE id = ?`,
        attachmentID)
    return err
}

// GetCachedAttachmentContent retorna conteúdo do cache
func GetCachedAttachmentContent(attachmentID int64) ([]byte, bool, error) {
    var cache struct {
        Data       []byte `db:"data"`
        Compressed bool   `db:"compressed"`
    }
    err := db.Get(&cache, `
        SELECT data, compressed FROM attachment_cache WHERE attachment_id = ?`,
        attachmentID)
    if err != nil {
        return nil, false, err
    }

    // Atualiza last_accessed
    db.Exec(`UPDATE attachment_cache SET last_accessed = CURRENT_TIMESTAMP WHERE attachment_id = ?`, attachmentID)

    return cache.Data, cache.Compressed, nil
}

// CleanupOldCache remove anexos cacheados não acessados há mais de N dias
func CleanupOldCache(olderThanDays int) (int64, error) {
    result, err := db.Exec(`
        DELETE FROM attachment_cache
        WHERE last_accessed < datetime('now', '-' || ? || ' days')`,
        olderThanDays)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}
```

---

## 4. Integração no TUI

### 4.1 Teclas e Comandos

| Tecla | Ação | Contexto |
|-------|------|----------|
| `A` | Abrir lista de anexos | Email viewer |
| `d` | Baixar anexo selecionado | Lista de anexos |
| `D` | Baixar todos anexos | Lista de anexos |
| `o` | Abrir anexo no app padrão | Lista de anexos |
| `s` | Salvar anexo (escolher local) | Lista de anexos |
| `Esc` | Fechar lista de anexos | Lista de anexos |
| `j/k` | Navegar entre anexos | Lista de anexos |

### 4.2 Modificações em `internal/tui/inbox/inbox.go`

```go
// Adicionar no Model struct:
type Model struct {
    // ... campos existentes ...

    // Attachments panel
    showAttachments       bool
    attachmentsList       []storage.AttachmentSummary
    selectedAttachment    int
    attachmentDownloading bool
}

// Adicionar messages:
type attachmentsLoadedMsg struct {
    attachments []storage.AttachmentSummary
    err         error
}

type attachmentDownloadedMsg struct {
    attachmentID int64
    data         []byte
    filename     string
    err          error
}
```

### 4.3 View do Painel de Anexos

```go
func (m Model) renderAttachmentsPanel() string {
    var sb strings.Builder

    sb.WriteString(boxStyle.Render("ANEXOS"))
    sb.WriteString("\n\n")

    if len(m.attachmentsList) == 0 {
        sb.WriteString(subtitleStyle.Render("Nenhum anexo neste email"))
        return sb.String()
    }

    for i, att := range m.attachmentsList {
        var style = readStyle
        if i == m.selectedAttachment {
            style = selectedStyle
        }

        var icon = getAttachmentIcon(att.ContentType)
        var sizeStr = formatSize(att.Size)
        var cached = ""
        if att.IsCached {
            cached = " [cached]"
        }

        var line = fmt.Sprintf("%s %s (%s)%s", icon, att.Filename, sizeStr, cached)
        sb.WriteString(style.Render(line))
        sb.WriteString("\n")
    }

    sb.WriteString("\n")
    sb.WriteString(subtitleStyle.Render("[d]ownload [o]pen [s]ave [D]ownload all [Esc]close"))

    return sb.String()
}

func getAttachmentIcon(contentType string) string {
    switch {
    case strings.HasPrefix(contentType, "image/"):
        return "🖼️"
    case strings.HasPrefix(contentType, "audio/"):
        return "🎵"
    case strings.HasPrefix(contentType, "video/"):
        return "🎬"
    case strings.Contains(contentType, "pdf"):
        return "📄"
    case strings.Contains(contentType, "zip") || strings.Contains(contentType, "rar"):
        return "📦"
    case strings.Contains(contentType, "spreadsheet") || strings.Contains(contentType, "excel"):
        return "📊"
    case strings.Contains(contentType, "document") || strings.Contains(contentType, "word"):
        return "📝"
    default:
        return "📎"
    }
}
```

---

## 5. Integração no Desktop

### 5.1 Novos Bindings em `internal/desktop/bindings.go`

```go
// GetAttachments returns attachments for an email
func (a *App) GetAttachments(emailID int64) ([]AttachmentDTO, error) {
    var attachments, err = storage.GetAttachmentsByEmail(emailID)
    if err != nil {
        return nil, err
    }

    var result []AttachmentDTO
    for _, att := range attachments {
        result = append(result, AttachmentDTO{
            ID:          att.ID,
            Filename:    att.Filename,
            ContentType: att.ContentType,
            Size:        att.Size,
            IsInline:    att.IsInline,
        })
    }
    return result, nil
}

// DownloadAttachment baixa um anexo para ~/Downloads
func (a *App) DownloadAttachment(attachmentID int64) (string, error) {
    // 1. Buscar metadados do anexo
    // 2. Verificar se está em cache
    // 3. Se não, buscar via IMAP usando PartNumber
    // 4. Salvar em ~/Downloads
    // 5. Retornar caminho do arquivo
}

// OpenAttachment abre anexo com app padrão
func (a *App) OpenAttachment(attachmentID int64) error {
    // 1. Baixar se necessário
    // 2. xdg-open (Linux) / open (macOS) / start (Windows)
}
```

### 5.2 Componente Svelte para Anexos

```svelte
<!-- AttachmentList.svelte -->
<script>
  export let attachments = [];

  async function downloadAttachment(att) {
    try {
      var path = await window.go.desktop.App.DownloadAttachment(att.id);
      // Show success notification
    } catch (err) {
      console.error('Download failed:', err);
    }
  }

  function getIcon(contentType) {
    if (contentType.startsWith('image/')) return '🖼️';
    if (contentType.startsWith('audio/')) return '🎵';
    if (contentType.startsWith('video/')) return '🎬';
    if (contentType.includes('pdf')) return '📄';
    return '📎';
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1024 / 1024).toFixed(1) + ' MB';
  }
</script>

{#if attachments.length > 0}
<div class="attachments-panel">
  <h4>📎 Anexos ({attachments.length})</h4>
  <ul>
    {#each attachments as att}
      <li class="attachment-item">
        <span class="icon">{getIcon(att.contentType)}</span>
        <span class="name">{att.filename}</span>
        <span class="size">({formatSize(att.size)})</span>
        <button on:click={() => downloadAttachment(att)} title="Download">⬇️</button>
      </li>
    {/each}
  </ul>
</div>
{/if}
```

---

## 6. Considerações de Performance

### 6.1 Download Sob Demanda vs Eager

| Estratégia | Quando Usar | Pros | Cons |
|------------|-------------|------|------|
| **Sob demanda** | Default | Economia de banda e storage | Latência ao visualizar |
| **Eager (inline)** | Imagens inline < 100KB | Display instantâneo | Sync mais lento |
| **Cache local** | Anexos acessados 2+ vezes | Balance | Gestão de espaço |

### 6.2 Constantes Recomendadas

```go
const (
    MaxInlineCacheSize = 100 * 1024      // 100KB - cachear automaticamente
    MaxAttachmentSize  = 25 * 1024 * 1024 // 25MB - limite de download
    CacheMaxDays       = 30               // Dias para manter cache
    CacheMaxTotalMB    = 500              // Limite total de cache
)
```

### 6.3 Estratégia de Cache

1. **Imagens inline < 100KB**: cachear automaticamente no sync
2. **Outros anexos**: baixar sob demanda
3. **Limpar cache**: FIFO quando > 500MB ou > 30 dias sem acesso

---

## 7. Considerações de Segurança

### 7.1 Tipos Permitidos

```go
var AllowedAttachmentTypes = map[string]bool{
    // Imagens
    "image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
    "image/svg+xml": false, // SVG pode conter scripts

    // Documentos
    "application/pdf": true,
    "application/msword": true,
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
    "application/vnd.ms-excel": true,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
    "text/plain": true,
    "text/csv": true,

    // Arquivos compactados
    "application/zip": true,
    "application/x-rar-compressed": true,
    "application/gzip": true,

    // Audio/Video
    "audio/mpeg": true, "audio/wav": true,
    "video/mp4": true, "video/webm": true,
}
```

### 7.2 Extensões BLOQUEADAS

```go
var BlockedExtensions = []string{
    ".exe", ".bat", ".cmd", ".com", ".scr", ".pif",
    ".vbs", ".vbe", ".js", ".jse", ".ws", ".wsf",
    ".msi", ".msp", ".dll", ".cpl",
    ".hta", ".jar", ".ps1", ".psm1",
}
```

### 7.3 Sanitização de Filename

```go
func SanitizeFilename(filename string) string {
    // Remover path traversal
    filename = filepath.Base(filename)

    // Remover caracteres perigosos
    var invalid = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
    filename = invalid.ReplaceAllString(filename, "_")

    // Limitar tamanho
    if len(filename) > 200 {
        var ext = filepath.Ext(filename)
        filename = filename[:200-len(ext)] + ext
    }

    return filename
}
```

---

## 8. Fluxo de Implementação Recomendado

### Fase 1: Schema e Storage (1-2 dias)
- [ ] Adicionar schema SQL em `db.go`
- [ ] Criar migração para adicionar tabelas
- [ ] Implementar models em `models.go`
- [ ] Implementar repository em `repository.go`
- [ ] Testes unitários

### Fase 2: IMAP Integration (1-2 dias)
- [ ] Adicionar `FetchAttachmentMetadata` no IMAP client
- [ ] Adicionar `FetchAttachmentPart` para download de partes
- [ ] Modificar `FetchEmailRaw` para popular metadados de anexos
- [ ] Modificar sync para extrair e salvar metadados

### Fase 3: Service Layer (1 dia)
- [ ] Criar `AttachmentService` em `internal/services/attachment.go`
- [ ] Adicionar interface em `ports/`
- [ ] Implementar download sob demanda
- [ ] Implementar cache de inline images

### Fase 4: TUI Integration (1-2 dias)
- [ ] Adicionar overlay de lista de anexos
- [ ] Implementar navegação e download
- [ ] Integrar com preview de imagens existente
- [ ] Adicionar indicador visual de anexos na lista de emails

### Fase 5: Desktop Integration (1 dia)
- [ ] Adicionar bindings para attachments
- [ ] Criar/atualizar componente Svelte
- [ ] Implementar preview e download
- [ ] Drag-and-drop para anexos (bonus)

### Fase 6: Polish (1 dia)
- [ ] Progress bar para downloads grandes
- [ ] Cleanup automático de cache
- [ ] Estatísticas de storage
- [ ] Documentação

---

## 9. Arquivos Críticos para Implementação

| Arquivo | Modificação |
|---------|-------------|
| `internal/storage/db.go` | Schema SQL e migração |
| `internal/storage/models.go` | Structs Attachment e AttachmentSummary |
| `internal/storage/repository.go` | Funções CRUD para attachments |
| `internal/imap/client.go` | FetchAttachmentMetadata e FetchAttachmentPart |
| `internal/tui/inbox/inbox.go` | UI para listar/baixar anexos |
| `internal/desktop/bindings.go` | Bindings GetAttachments, DownloadAttachment |
| `cmd/miau-desktop/frontend/src/lib/components/` | Componente Svelte |

---

## 10. Testes Necessários

```go
// internal/storage/attachments_test.go
func TestUpsertAttachment(t *testing.T) { /* ... */ }
func TestGetAttachmentsByEmail(t *testing.T) { /* ... */ }
func TestCacheAttachmentContent(t *testing.T) { /* ... */ }
func TestCleanupOldCache(t *testing.T) { /* ... */ }

// internal/imap/client_test.go
func TestFetchAttachmentMetadata(t *testing.T) { /* ... */ }
func TestFetchAttachmentPart(t *testing.T) { /* ... */ }

// internal/services/attachment_test.go
func TestDownloadAttachment(t *testing.T) { /* ... */ }
func TestValidateAttachment(t *testing.T) { /* ... */ }
func TestSanitizeFilename(t *testing.T) { /* ... */ }
```

---

*Última atualização: 2024-12-04*
