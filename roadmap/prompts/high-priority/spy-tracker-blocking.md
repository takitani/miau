# Prompt: Spy Tracker Blocking (Bloqueio de Rastreadores)

> Inspirado no HEY.com - Bloquear pixels de tracking invisíveis.

## Conceito

Muitos emails incluem "spy pixels" - imagens invisíveis de 1x1 pixel que notificam o remetente quando você abre o email. Isso viola sua privacidade.

```
┌─ Email de marketing@empresa.com ─────────────────────────────┐
│                                                              │
│  🛡️ 2 rastreadores bloqueados                                │
│  ├─ mailchimp.com (pixel de abertura)                       │
│  └─ analytics.google.com (tracking)                          │
│                                                              │
│  Assunto: Oferta especial para você!                        │
│  ────────────────────────────────────                        │
│  [conteúdo do email sem rastreadores]                        │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Tipos de Rastreadores

### 1. Pixel de Abertura (Open Tracking)
```html
<!-- Imagem 1x1 com ID único -->
<img src="https://track.mailchimp.com/open?id=abc123" width="1" height="1">
```

### 2. Link Tracking
```html
<!-- Link que passa por redirect -->
<a href="https://click.mailchimp.com/redirect?url=https://site.com&id=abc123">
  Clique aqui
</a>
```

### 3. Web Beacons
```html
<!-- CSS que carrega recurso externo -->
<style>
  body { background: url('https://tracker.com/beacon?id=123'); }
</style>
```

## Database Schema

```sql
-- Domínios conhecidos de tracking
CREATE TABLE tracking_domains (
    id INTEGER PRIMARY KEY,
    domain TEXT UNIQUE NOT NULL,
    tracker_type TEXT NOT NULL,    -- pixel, link, beacon
    company TEXT,                  -- Mailchimp, Google, etc
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Log de rastreadores bloqueados
CREATE TABLE blocked_trackers (
    id INTEGER PRIMARY KEY,
    email_id INTEGER NOT NULL REFERENCES emails(id),
    tracker_url TEXT NOT NULL,
    tracker_type TEXT NOT NULL,
    domain TEXT NOT NULL,
    blocked_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Configuração do usuário
CREATE TABLE tracker_settings (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    block_pixels BOOLEAN DEFAULT 1,
    block_link_tracking BOOLEAN DEFAULT 0,  -- Mais agressivo
    show_badge BOOLEAN DEFAULT 1,
    whitelist TEXT,                          -- JSON array de domínios permitidos
    UNIQUE(account_id)
);

-- Index
CREATE INDEX idx_blocked_email ON blocked_trackers(email_id);
```

## Domínios de Tracking Conhecidos

```go
// internal/services/tracker/domains.go

var knownTrackingDomains = map[string]TrackerInfo{
    // Email Marketing
    "mailchimp.com":           {Type: "pixel", Company: "Mailchimp"},
    "list-manage.com":         {Type: "pixel", Company: "Mailchimp"},
    "sendgrid.net":            {Type: "pixel", Company: "SendGrid"},
    "sendgrid.com":            {Type: "pixel", Company: "SendGrid"},
    "amazonses.com":           {Type: "pixel", Company: "Amazon SES"},
    "mailgun.org":             {Type: "pixel", Company: "Mailgun"},
    "constantcontact.com":     {Type: "pixel", Company: "Constant Contact"},
    "hubspot.com":             {Type: "pixel", Company: "HubSpot"},
    "hubspotemail.net":        {Type: "pixel", Company: "HubSpot"},
    "mailerlite.com":          {Type: "pixel", Company: "MailerLite"},
    "convertkit.com":          {Type: "pixel", Company: "ConvertKit"},
    "drip.com":                {Type: "pixel", Company: "Drip"},
    "getresponse.com":         {Type: "pixel", Company: "GetResponse"},
    "aweber.com":              {Type: "pixel", Company: "AWeber"},
    "infusionsoft.com":        {Type: "pixel", Company: "Keap"},
    "activecampaign.com":      {Type: "pixel", Company: "ActiveCampaign"},
    "klaviyo.com":             {Type: "pixel", Company: "Klaviyo"},
    "intercom.io":             {Type: "pixel", Company: "Intercom"},
    "customer.io":             {Type: "pixel", Company: "Customer.io"},
    "mixpanel.com":            {Type: "pixel", Company: "Mixpanel"},
    "segment.io":              {Type: "pixel", Company: "Segment"},

    // Analytics
    "google-analytics.com":    {Type: "pixel", Company: "Google"},
    "googleusercontent.com":   {Type: "pixel", Company: "Google"},
    "doubleclick.net":         {Type: "pixel", Company: "Google"},
    "facebook.com":            {Type: "pixel", Company: "Facebook"},
    "linkedin.com":            {Type: "pixel", Company: "LinkedIn"},

    // Superhuman tracking
    "superhuman.com":          {Type: "pixel", Company: "Superhuman"},

    // Link shorteners (tracking via redirect)
    "bit.ly":                  {Type: "link", Company: "Bitly"},
    "t.co":                    {Type: "link", Company: "Twitter"},
    "goo.gl":                  {Type: "link", Company: "Google"},
    "ow.ly":                   {Type: "link", Company: "Hootsuite"},
    "tinyurl.com":             {Type: "link", Company: "TinyURL"},
}
```

## Service Implementation

```go
// internal/services/tracker.go

type TrackerBlockerService struct {
    storage ports.TrackerStorage
    domains map[string]TrackerInfo
}

type TrackerInfo struct {
    Type    string // pixel, link, beacon
    Company string
}

type BlockedTracker struct {
    URL     string
    Type    string
    Domain  string
    Company string
}

// SanitizeEmailHTML remove rastreadores do HTML
func (s *TrackerBlockerService) SanitizeEmailHTML(ctx context.Context, emailID int64, html string) (string, []BlockedTracker, error) {
    blocked := make([]BlockedTracker, 0)

    // Parse HTML
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return html, nil, err
    }

    // 1. Remover pixels de imagem
    doc.Find("img").Each(func(i int, img *goquery.Selection) {
        src, exists := img.Attr("src")
        if !exists {
            return
        }

        // Verificar se é pixel de tracking
        if s.isTrackingPixel(img, src) {
            domain := s.extractDomain(src)
            info := s.domains[domain]

            blocked = append(blocked, BlockedTracker{
                URL:     src,
                Type:    "pixel",
                Domain:  domain,
                Company: info.Company,
            })

            // Remover ou substituir por placeholder
            img.Remove()
        }
    })

    // 2. Remover web beacons em CSS
    doc.Find("style").Each(func(i int, style *goquery.Selection) {
        css := style.Text()
        newCSS, beacons := s.sanitizeCSS(css)
        blocked = append(blocked, beacons...)
        style.SetText(newCSS)
    })

    // 3. Remover atributos style com tracking
    doc.Find("[style]").Each(func(i int, el *goquery.Selection) {
        style, _ := el.Attr("style")
        newStyle, beacons := s.sanitizeInlineStyle(style)
        blocked = append(blocked, beacons...)
        el.SetAttr("style", newStyle)
    })

    // Salvar log de bloqueados
    if len(blocked) > 0 {
        for _, b := range blocked {
            s.storage.LogBlocked(ctx, emailID, b)
        }
    }

    // Retornar HTML sanitizado
    result, _ := doc.Html()
    return result, blocked, nil
}

func (s *TrackerBlockerService) isTrackingPixel(img *goquery.Selection, src string) bool {
    // 1. Verificar domínio conhecido
    domain := s.extractDomain(src)
    if _, known := s.domains[domain]; known {
        return true
    }

    // 2. Verificar dimensões (1x1 ou muito pequeno)
    width, _ := img.Attr("width")
    height, _ := img.Attr("height")
    if width == "1" || height == "1" || width == "0" || height == "0" {
        return true
    }

    // 3. Verificar padrões comuns de URL
    trackingPatterns := []string{
        "/open", "/track", "/pixel", "/beacon",
        "tracking", "analytics", "/o/", "/t/",
        "?id=", "&uid=", "?u=", "?email=",
    }
    srcLower := strings.ToLower(src)
    for _, pattern := range trackingPatterns {
        if strings.Contains(srcLower, pattern) {
            return true
        }
    }

    // 4. Verificar se é imagem transparente base64
    if strings.Contains(src, "data:image") && len(src) < 200 {
        return true // Base64 muito curto = provavelmente pixel
    }

    return false
}

func (s *TrackerBlockerService) sanitizeCSS(css string) (string, []BlockedTracker) {
    blocked := make([]BlockedTracker, 0)

    // Regex para encontrar url() em CSS
    urlPattern := regexp.MustCompile(`url\(['"]?(https?://[^'")\s]+)['"]?\)`)

    result := urlPattern.ReplaceAllStringFunc(css, func(match string) string {
        submatch := urlPattern.FindStringSubmatch(match)
        if len(submatch) < 2 {
            return match
        }

        url := submatch[1]
        domain := s.extractDomain(url)

        if _, known := s.domains[domain]; known {
            blocked = append(blocked, BlockedTracker{
                URL:     url,
                Type:    "beacon",
                Domain:  domain,
                Company: s.domains[domain].Company,
            })
            return "url()" // Remover URL
        }

        return match
    })

    return result, blocked
}

// GetBlockedTrackers retorna rastreadores bloqueados de um email
func (s *TrackerBlockerService) GetBlockedTrackers(ctx context.Context, emailID int64) ([]BlockedTracker, error) {
    return s.storage.GetBlocked(ctx, emailID)
}

// GetStats retorna estatísticas de bloqueio
func (s *TrackerBlockerService) GetStats(ctx context.Context, accountID int64) (*TrackerStats, error) {
    return s.storage.GetStats(ctx, accountID)
}

type TrackerStats struct {
    TotalBlocked     int
    UniqueEmails     int
    TopCompanies     []CompanyCount
    BlockedLastWeek  int
    BlockedLastMonth int
}
```

## Desktop UI

```svelte
<!-- TrackerBadge.svelte -->
<script>
  import { GetBlockedTrackers } from '../wailsjs/go/desktop/App';

  export let emailId;

  let trackers = [];
  let showDetails = false;

  $: if (emailId) {
    loadTrackers();
  }

  async function loadTrackers() {
    trackers = await GetBlockedTrackers(emailId);
  }
</script>

{#if trackers.length > 0}
  <button
    class="tracker-badge"
    on:click={() => showDetails = !showDetails}
    title="Clique para ver detalhes"
  >
    <span class="shield">🛡️</span>
    <span class="count">{trackers.length} rastreador{trackers.length > 1 ? 'es' : ''} bloqueado{trackers.length > 1 ? 's' : ''}</span>
  </button>

  {#if showDetails}
    <div class="tracker-details">
      <h4>Rastreadores bloqueados neste email:</h4>
      <ul>
        {#each trackers as tracker}
          <li>
            <span class="domain">{tracker.domain}</span>
            {#if tracker.company}
              <span class="company">({tracker.company})</span>
            {/if}
            <span class="type">{tracker.type}</span>
          </li>
        {/each}
      </ul>
      <p class="info">
        Estes rastreadores foram removidos para proteger sua privacidade.
        Sem eles, o remetente não sabe quando você abriu o email.
      </p>
    </div>
  {/if}
{/if}

<style>
  .tracker-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-warning);
    color: var(--text-warning);
    border: none;
    border-radius: var(--radius-sm);
    font-size: 0.85em;
    cursor: pointer;
  }

  .tracker-badge:hover {
    background: var(--bg-warning-hover);
  }

  .shield {
    font-size: 1.1em;
  }

  .tracker-details {
    margin-top: var(--space-sm);
    padding: var(--space-md);
    background: var(--bg-secondary);
    border-radius: var(--radius-md);
    font-size: 0.9em;
  }

  .tracker-details h4 {
    margin: 0 0 var(--space-sm);
    font-size: 0.95em;
  }

  .tracker-details ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .tracker-details li {
    padding: var(--space-xs) 0;
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .domain {
    font-family: monospace;
    color: var(--text-primary);
  }

  .company {
    color: var(--text-secondary);
  }

  .type {
    font-size: 0.8em;
    padding: 2px 6px;
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    color: var(--text-tertiary);
  }

  .info {
    margin-top: var(--space-md);
    padding-top: var(--space-md);
    border-top: 1px solid var(--border-subtle);
    color: var(--text-secondary);
    font-size: 0.85em;
  }
</style>
```

### Integração no EmailViewer

```svelte
<!-- EmailViewer.svelte -->
<script>
  import TrackerBadge from './TrackerBadge.svelte';

  // No header do email
</script>

<div class="email-header">
  <div class="from">{email.fromName}</div>
  <div class="subject">{email.subject}</div>
  <TrackerBadge emailId={email.id} />
</div>
```

## TUI Implementation

```go
// Mostrar badge no viewer
func (m *ViewerModel) renderHeader() string {
    var b strings.Builder

    b.WriteString(fmt.Sprintf("De: %s\n", m.email.FromName))
    b.WriteString(fmt.Sprintf("Assunto: %s\n", m.email.Subject))

    // Badge de trackers
    if len(m.blockedTrackers) > 0 {
        badge := fmt.Sprintf("🛡️ %d rastreador(es) bloqueado(s)", len(m.blockedTrackers))
        b.WriteString(m.styles.Warning.Render(badge) + "\n")
    }

    return b.String()
}

// Tecla 't' para ver detalhes
case "t":
    if len(m.blockedTrackers) > 0 {
        return m.showTrackerDetails()
    }
```

## Estatísticas de Privacidade

```svelte
<!-- PrivacyStats.svelte (no Settings ou Analytics) -->
<script>
  import { GetTrackerStats } from '../wailsjs/go/desktop/App';

  let stats = null;

  onMount(async () => {
    stats = await GetTrackerStats();
  });
</script>

{#if stats}
  <div class="privacy-stats">
    <h3>🛡️ Proteção de Privacidade</h3>

    <div class="stat-grid">
      <div class="stat">
        <div class="value">{stats.totalBlocked}</div>
        <div class="label">Rastreadores bloqueados</div>
      </div>

      <div class="stat">
        <div class="value">{stats.uniqueEmails}</div>
        <div class="label">Emails protegidos</div>
      </div>

      <div class="stat">
        <div class="value">{stats.blockedLastWeek}</div>
        <div class="label">Últimos 7 dias</div>
      </div>
    </div>

    <h4>Top empresas de tracking:</h4>
    <ol class="top-companies">
      {#each stats.topCompanies as company}
        <li>
          <span class="name">{company.name}</span>
          <span class="count">{company.count}</span>
        </li>
      {/each}
    </ol>
  </div>
{/if}
```

## Configurações

```svelte
<!-- Settings: Seção de Privacidade -->
<section class="privacy-settings">
  <h3>🛡️ Privacidade</h3>

  <label class="setting">
    <input type="checkbox" bind:checked={settings.blockPixels} />
    <div>
      <strong>Bloquear pixels de rastreamento</strong>
      <p>Remove imagens invisíveis que notificam quando você abre emails</p>
    </div>
  </label>

  <label class="setting">
    <input type="checkbox" bind:checked={settings.showBadge} />
    <div>
      <strong>Mostrar badge de rastreadores</strong>
      <p>Exibe quantos rastreadores foram bloqueados em cada email</p>
    </div>
  </label>

  <label class="setting">
    <input type="checkbox" bind:checked={settings.blockLinkTracking} />
    <div>
      <strong>Bloquear tracking de links (avançado)</strong>
      <p>Remove parâmetros de rastreamento de links. Pode quebrar alguns links.</p>
    </div>
  </label>
</section>
```

## Critérios de Aceitação

- [ ] Pixels de 1x1 são removidos
- [ ] Domínios conhecidos são bloqueados
- [ ] Badge mostra quantidade bloqueada
- [ ] Detalhes mostram empresa/tipo
- [ ] Estatísticas de privacidade disponíveis
- [ ] Configuração para habilitar/desabilitar
- [ ] HTML ainda renderiza corretamente após sanitização
- [ ] Performance: sanitização < 50ms

---

*Inspirado em: HEY.com Spy Tracker Blocking*
