<script>
  import { onMount } from 'svelte';
  import DOMPurify from 'dompurify';
  import { archiveEmail, deleteEmail, toggleStar, markAsRead } from '../stores/emails.js';
  import { showCompose } from '../stores/ui.js';

  export let email;

  // Configure DOMPurify for email-safe HTML
  // This is the industry-standard approach used by professional email clients
  var DOMPURIFY_CONFIG = {
    // Allowed tags - basic HTML structure + formatting
    ALLOWED_TAGS: [
      'a', 'abbr', 'acronym', 'address', 'area', 'article', 'aside',
      'b', 'bdi', 'bdo', 'big', 'blockquote', 'br',
      'caption', 'center', 'cite', 'code', 'col', 'colgroup',
      'data', 'dd', 'del', 'details', 'dfn', 'dir', 'div', 'dl', 'dt',
      'em',
      'figcaption', 'figure', 'font', 'footer',
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'header', 'hgroup', 'hr',
      'i', 'img', 'ins',
      'kbd',
      'li',
      'main', 'map', 'mark', 'menu', 'meter',
      'nav',
      'ol',
      'p', 'pre', 'progress',
      'q',
      'rp', 'rt', 'ruby',
      's', 'samp', 'section', 'small', 'span', 'strike', 'strong', 'sub', 'summary', 'sup',
      'table', 'tbody', 'td', 'tfoot', 'th', 'thead', 'time', 'tr', 'tt',
      'u', 'ul',
      'var', 'wbr'
    ],
    // Allowed attributes
    ALLOWED_ATTR: [
      'align', 'alt', 'bgcolor', 'border', 'cellpadding', 'cellspacing',
      'class', 'color', 'cols', 'colspan', 'coords', 'dir', 'disabled',
      'height', 'href', 'hspace', 'id', 'lang', 'name', 'noshade', 'nowrap',
      'rel', 'rows', 'rowspan', 'rules', 'scope', 'shape', 'size',
      'span', 'src', 'start', 'style', 'summary', 'tabindex', 'target',
      'title', 'type', 'usemap', 'valign', 'value', 'vspace', 'width',
      // Data attributes for our image blocking
      'data-blocked-src'
    ],
    // Block dangerous elements completely (strip content too)
    FORBID_TAGS: ['script', 'style', 'meta', 'link', 'base', 'object', 'embed', 'applet', 'frame', 'frameset', 'iframe'],
    // Block dangerous attributes
    FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'],
    // Strip data: URIs for src (except images we explicitly allow)
    ALLOW_DATA_ATTR: true,
    // Force all links to open in new tab for security
    ADD_ATTR: ['target'],
  };

  // Hook to force external links to open in new tab
  DOMPurify.addHook('afterSanitizeAttributes', function(node) {
    if (node.tagName === 'A' && node.hasAttribute('href')) {
      var href = node.getAttribute('href');
      // External links open in new tab
      if (href && (href.startsWith('http://') || href.startsWith('https://'))) {
        node.setAttribute('target', '_blank');
        node.setAttribute('rel', 'noopener noreferrer');
      }
    }
  });

  // Full email content (loaded on demand)
  let fullEmail = null;
  let loading = false;
  let showImages = false; // User must opt-in to load external images
  let hasExternalImages = false;

  // AI Summary state
  let summary = null;
  let summaryLoading = false;
  let summaryError = null;
  let showSummary = false;
  let summaryStyle = 'brief'; // 'tldr', 'brief', 'detailed'

  // Load full email when email changes
  $: if (email && email.id) {
    loadFullEmail(email.id);
    // Reset summary state when email changes
    summary = null;
    summaryError = null;
    showSummary = false;
    // Try to load cached summary
    loadCachedSummary(email.id);
  }

  async function loadFullEmail(id) {
    loading = true;
    showImages = false; // Reset for each email
    hasExternalImages = false;
    fullEmail = null;
    try {
      if (window.go?.desktop?.App) {
        fullEmail = await window.go.desktop.App.GetEmail(id);
        console.log('Loaded email:', fullEmail?.subject, 'bodyHtml length:', fullEmail?.bodyHtml?.length);
      } else {
        // Mock for development
        fullEmail = {
          ...email,
          toAddresses: 'me@example.com',
          ccAddresses: '',
          bodyText: 'This is the email body content.\n\nBest regards,\nSender',
          bodyHtml: '<div><p>Test HTML</p><img src="https://via.placeholder.com/150" /><br><div style="color: #666;">-- <br>Assinatura de teste</div></div>'
        };
      }
    } catch (err) {
      console.error('Failed to load email:', err);
    } finally {
      loading = false;
    }
  }

  // Load cached summary if exists
  async function loadCachedSummary(id) {
    try {
      if (window.go?.desktop?.App) {
        const cached = await window.go.desktop.App.GetCachedSummary(id);
        if (cached) {
          summary = cached;
          showSummary = true;
        }
      }
    } catch (err) {
      // Ignore errors - cache miss is expected
    }
  }

  // Generate AI summary
  async function generateSummary() {
    if (!email?.id) return;

    summaryLoading = true;
    summaryError = null;

    try {
      if (window.go?.desktop?.App) {
        summary = await window.go.desktop.App.SummarizeEmailWithStyle(email.id, summaryStyle);
        showSummary = true;
      }
    } catch (err) {
      summaryError = err.message || 'Erro ao gerar resumo';
      console.error('Failed to generate summary:', err);
    } finally {
      summaryLoading = false;
    }
  }

  // Change summary style and regenerate
  async function changeSummaryStyle(newStyle) {
    summaryStyle = newStyle;
    await generateSummary();
  }

  // Toggle summary visibility
  function toggleSummary() {
    if (!showSummary && !summary) {
      generateSummary();
    } else {
      showSummary = !showSummary;
    }
  }

  // Refresh summary (invalidate cache and regenerate)
  async function refreshSummary() {
    if (!email?.id) return;

    try {
      if (window.go?.desktop?.App) {
        await window.go.desktop.App.InvalidateSummary(email.id);
      }
    } catch (err) {
      // Ignore invalidation errors
    }
    await generateSummary();
  }

  // Process HTML for display - strip scripts, block external images
  let processedHtml = '';

  $: if (fullEmail?.bodyHtml) {
    processHtml(fullEmail.bodyHtml);
  }

  function processHtml(html) {
    if (!html) {
      processedHtml = '';
      return;
    }

    // Check if there are external images (http/https URLs, not data: or cid:)
    hasExternalImages = /src=["']https?:\/\//i.test(html);

    // Step 1: Convert cid: URLs to data: URLs BEFORE sanitization
    var processed = html;
    if (fullEmail?.attachments) {
      fullEmail.attachments.forEach(att => {
        if (att.isInline && att.contentId && att.data) {
          var cidPattern = new RegExp(`src=["']cid:${att.contentId.replace(/[<>]/g, '')}["']`, 'gi');
          var dataUrl = `src="data:${att.contentType};base64,${att.data}"`;
          processed = processed.replace(cidPattern, dataUrl);
          console.log(`[EmailViewer] Replaced cid:${att.contentId} with data URL`);
        }
      });
    }

    // Step 2: Block external images BEFORE sanitization (to preserve data-blocked-src)
    if (!showImages && hasExternalImages) {
      processed = processed.replace(
        /<img([^>]*)src=["'](https?:\/\/[^"']+)["']([^>]*)>/gi,
        '<img$1src="data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'120\' height=\'80\'%3E%3Crect fill=\'%232a2a3e\' width=\'120\' height=\'80\' rx=\'4\'/%3E%3Ctext x=\'60\' y=\'45\' fill=\'%23888\' text-anchor=\'middle\' font-size=\'11\'%3EImagem bloqueada%3C/text%3E%3C/svg%3E" data-blocked-src="$2"$3 style="cursor:pointer;border:1px dashed #444;" title="Imagem externa bloqueada">'
      );
    }

    // Step 3: Sanitize with DOMPurify - industry standard for email HTML
    // This removes ALL dangerous content: scripts, meta tags, event handlers, etc.
    var safe = DOMPurify.sanitize(processed, DOMPURIFY_CONFIG);

    console.log('[EmailViewer] Sanitized HTML with DOMPurify');
    processedHtml = safe;
  }

  function loadExternalImages() {
    showImages = true;
    if (fullEmail?.bodyHtml) {
      processHtml(fullEmail.bodyHtml);
    }
  }

  // Format date
  function formatDate(dateStr) {
    var date = new Date(dateStr);
    return date.toLocaleDateString('pt-BR', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  // Handle actions
  function handleArchive() {
    archiveEmail(email.id);
  }

  function handleDelete() {
    deleteEmail(email.id);
  }

  function handleStar() {
    toggleStar(email.id);
  }

  function handleMarkUnread() {
    markAsRead(email.id, false);
  }

  function handleReply() {
    window.composeContext = {
      mode: 'reply',
      replyTo: fullEmail || email
    };
    showCompose.set(true);
  }

  function handleReplyAll() {
    window.composeContext = {
      mode: 'replyAll',
      replyTo: fullEmail || email
    };
    showCompose.set(true);
  }

  function handleForward() {
    window.composeContext = {
      mode: 'forward',
      forwardEmail: fullEmail || email
    };
    showCompose.set(true);
  }

  // Attachment helpers
  function getAttachmentIcon(contentType) {
    if (!contentType) return '📄';
    if (contentType.startsWith('image/')) return '🖼️';
    if (contentType.startsWith('video/')) return '🎬';
    if (contentType.startsWith('audio/')) return '🎵';
    if (contentType.includes('pdf')) return '📕';
    if (contentType.includes('word') || contentType.includes('document')) return '📘';
    if (contentType.includes('excel') || contentType.includes('sheet')) return '📗';
    if (contentType.includes('zip') || contentType.includes('compressed')) return '📦';
    if (contentType.includes('text')) return '📝';
    return '📄';
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return Math.round(bytes / 1024) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  async function downloadAttachment(att) {
    try {
      if (window.go?.desktop?.App) {
        // Use email ID and part number to download
        var result = await window.go.desktop.App.SaveAttachmentByPart(email.id, att.partNumber, att.filename);
        if (result) {
          console.log('Attachment saved to:', result);
        }
      }
    } catch (err) {
      console.error('Failed to download attachment:', err);
      alert('Erro ao baixar anexo: ' + err.message);
    }
  }

  async function openAttachment(att) {
    try {
      if (window.go?.desktop?.App) {
        // Open with default application using email ID and part number
        await window.go.desktop.App.OpenAttachmentByPart(email.id, att.partNumber, att.filename);
      }
    } catch (err) {
      console.error('Failed to open attachment:', err);
      alert('Erro ao abrir anexo: ' + err.message);
    }
  }
</script>

<div class="email-viewer">
  {#if loading}
    <div class="loading">
      <span class="spinner"></span>
      Carregando...
    </div>
  {:else if email}
    <!-- Toolbar -->
    <div class="toolbar">
      <div class="toolbar-group">
        <button class="action primary" title="Responder (r)" on:click={handleReply}>
          ↩️ Responder
        </button>
        <button class="action" title="Responder Todos (R)" on:click={handleReplyAll}>
          ↩️↩️ Todos
        </button>
        <button class="action" title="Encaminhar (f)" on:click={handleForward}>
          ➡️ Encaminhar
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-group">
        <button
          class="action"
          class:active={showSummary}
          title="Resumo IA (s)"
          on:click={toggleSummary}
          disabled={summaryLoading}
        >
          {#if summaryLoading}
            <span class="spinner-small"></span>
          {:else}
            🤖
          {/if}
          Resumo
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-group">
        <button class="action" title="Arquivar (e)" on:click={handleArchive}>
          📁
        </button>
        <button class="action" title="Excluir (x)" on:click={handleDelete}>
          🗑️
        </button>
        <button class="action" title="Estrela (s)" on:click={handleStar}>
          {email.isStarred ? '★' : '☆'}
        </button>
        <button class="action" title="Marcar não lido (u)" on:click={handleMarkUnread}>
          ✉️
        </button>
      </div>
    </div>

    <!-- Headers -->
    <header class="email-header">
      <h1 class="subject">{email.subject || '(sem assunto)'}</h1>

      <div class="meta">
        <div class="from">
          <strong>{email.fromName || email.fromEmail}</strong>
          {#if email.fromName}
            <span class="email">&lt;{email.fromEmail}&gt;</span>
          {/if}
        </div>

        <div class="date">{formatDate(email.date)}</div>
      </div>

      {#if fullEmail?.toAddresses}
        <div class="recipients">
          <span class="label">Para:</span>
          <span class="value">{fullEmail.toAddresses}</span>
        </div>
      {/if}

      {#if fullEmail?.ccAddresses}
        <div class="recipients">
          <span class="label">Cc:</span>
          <span class="value">{fullEmail.ccAddresses}</span>
        </div>
      {/if}
    </header>

    <!-- Image warning banner -->
    {#if hasExternalImages && !showImages}
      <div class="image-banner">
        <span>🖼️ Imagens externas bloqueadas por segurança</span>
        <button on:click={loadExternalImages}>Mostrar Imagens</button>
      </div>
    {/if}

    <!-- AI Summary Section -->
    {#if showSummary || summaryLoading || summaryError}
      <div class="summary-section">
        <div class="summary-header">
          <span class="summary-title">
            🤖 Resumo IA
            {#if summary?.cached}
              <span class="cached-badge" title="Carregado do cache">📦</span>
            {/if}
          </span>
          <div class="summary-controls">
            <select
              class="style-select"
              bind:value={summaryStyle}
              on:change={() => changeSummaryStyle(summaryStyle)}
              disabled={summaryLoading}
            >
              <option value="tldr">TL;DR (1-2 frases)</option>
              <option value="brief">Breve (3-5 frases)</option>
              <option value="detailed">Detalhado</option>
            </select>
            <button
              class="summary-btn"
              title="Atualizar resumo"
              on:click={refreshSummary}
              disabled={summaryLoading}
            >
              🔄
            </button>
            <button
              class="summary-btn"
              title="Fechar"
              on:click={() => showSummary = false}
            >
              ✕
            </button>
          </div>
        </div>

        {#if summaryLoading}
          <div class="summary-loading">
            <span class="spinner-small"></span>
            Gerando resumo...
          </div>
        {:else if summaryError}
          <div class="summary-error">
            ⚠️ {summaryError}
            <button class="retry-btn" on:click={generateSummary}>Tentar novamente</button>
          </div>
        {:else if summary}
          <div class="summary-content">
            <p>{summary.content}</p>
            {#if summary.keyPoints && summary.keyPoints.length > 0}
              <div class="key-points">
                <strong>Pontos-chave:</strong>
                <ul>
                  {#each summary.keyPoints as point}
                    <li>{point}</li>
                  {/each}
                </ul>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Body -->
    <div class="email-body">
      {#if fullEmail?.bodyHtml}
        <!-- Render HTML directly (scripts stripped, external images blocked) -->
        <div class="html-content">
          {@html processedHtml}
        </div>
      {:else if fullEmail?.bodyText}
        <!-- Render plain text -->
        <pre class="text-body">{fullEmail.bodyText}</pre>
      {:else}
        <p class="snippet">{email.snippet}</p>
      {/if}
    </div>

    <!-- Attachments -->
    {#if fullEmail?.attachments && fullEmail.attachments.length > 0}
      <div class="attachments">
        <h3>📎 Anexos ({fullEmail.attachments.length})</h3>
        <ul>
          {#each fullEmail.attachments as att}
            <li class="attachment">
              <span class="icon">{getAttachmentIcon(att.contentType)}</span>
              <span class="name">{att.filename}</span>
              <span class="size">({formatSize(att.size)})</span>
              <div class="attachment-actions">
                <button class="action-btn" on:click={() => openAttachment(att)} title="Abrir com aplicativo padrão">
                  ▶️ Abrir
                </button>
                <button class="action-btn" on:click={() => downloadAttachment(att)} title="Salvar em...">
                  💾 Salvar
                </button>
              </div>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  {/if}
</div>

<style>
  .email-viewer {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
  }

  .loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 200px;
    color: var(--text-muted);
  }

  .spinner {
    display: inline-block;
    width: 24px;
    height: 24px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: var(--space-sm);
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Toolbar */
  .toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-secondary);
  }

  .toolbar-group {
    display: flex;
    gap: var(--space-xs);
  }

  .toolbar-divider {
    width: 1px;
    height: 24px;
    background: var(--border-color);
    margin: 0 var(--space-xs);
  }

  .action {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-sm);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .action:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .action.primary {
    background: var(--accent-primary);
    color: white;
  }

  .action.primary:hover {
    background: var(--accent-secondary);
  }

  /* Image banner */
  .image-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-sm) var(--space-md);
    background: rgba(234, 179, 8, 0.15);
    border-bottom: 1px solid rgba(234, 179, 8, 0.3);
    font-size: var(--font-sm);
    color: #fbbf24;
  }

  .image-banner button {
    padding: var(--space-xs) var(--space-sm);
    background: rgba(234, 179, 8, 0.2);
    border: 1px solid rgba(234, 179, 8, 0.4);
    border-radius: var(--radius-sm);
    color: #fbbf24;
    font-size: var(--font-xs);
    cursor: pointer;
  }

  .image-banner button:hover {
    background: rgba(234, 179, 8, 0.3);
  }

  /* Header */
  .email-header {
    padding: var(--space-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .subject {
    font-size: var(--font-xl);
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: var(--space-md);
  }

  .meta {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: var(--space-sm);
  }

  .from {
    color: var(--text-primary);
  }

  .from .email {
    color: var(--text-muted);
    font-size: var(--font-sm);
    margin-left: var(--space-xs);
  }

  .date {
    color: var(--text-muted);
    font-size: var(--font-sm);
  }

  .recipients {
    font-size: var(--font-sm);
    color: var(--text-secondary);
    margin-top: var(--space-xs);
  }

  .recipients .label {
    color: var(--text-muted);
    margin-right: var(--space-xs);
  }

  /* Body */
  .email-body {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .html-content {
    flex: 1;
    width: 100%;
    padding: 16px;
    background: #1a1a2e;
    overflow-y: auto;
    color: #e0e0e0;
    font-size: 14px;
    line-height: 1.6;
    word-wrap: break-word;
  }

  .html-content :global(a) {
    color: #60a5fa;
  }

  .html-content :global(img) {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
  }

  .html-content :global(blockquote) {
    margin: 8px 0;
    padding-left: 12px;
    border-left: 3px solid #404040;
    color: #a0a0a0;
  }

  .html-content :global(pre),
  .html-content :global(code) {
    background: #2a2a3e;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: monospace;
  }

  .html-content :global(pre) {
    padding: 12px;
    overflow-x: auto;
  }

  .html-content :global(table) {
    border-collapse: collapse;
    max-width: 100%;
  }

  .html-content :global(td),
  .html-content :global(th) {
    border: 1px solid #404040;
    padding: 8px;
  }

  .html-content :global(hr) {
    border: none;
    border-top: 1px solid #404040;
    margin: 16px 0;
  }

  .text-body {
    white-space: pre-wrap;
    font-family: inherit;
    font-size: var(--font-md);
    line-height: 1.6;
    color: var(--text-primary);
    padding: var(--space-lg);
    overflow-y: auto;
    flex: 1;
  }

  .snippet {
    color: var(--text-secondary);
    font-style: italic;
    padding: var(--space-lg);
  }

  /* Attachments */
  .attachments {
    padding: var(--space-md);
    border-top: 1px solid var(--border-color);
    background: var(--bg-secondary);
  }

  .attachments h3 {
    font-size: var(--font-sm);
    font-weight: 600;
    color: var(--text-muted);
    margin-bottom: var(--space-sm);
  }

  .attachments ul {
    list-style: none;
  }

  .attachment {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm);
    font-size: var(--font-sm);
    background: var(--bg-primary);
    border-radius: var(--radius-sm);
    margin-bottom: var(--space-xs);
  }

  .attachment:hover {
    background: var(--bg-hover);
  }

  .attachment .icon {
    font-size: 1.2em;
  }

  .attachment .name {
    color: var(--accent-primary);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .attachment .size {
    color: var(--text-muted);
    font-size: var(--font-xs);
  }

  .attachment-actions {
    display: flex;
    gap: var(--space-xs);
  }

  .action-btn {
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-xs);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .action-btn:hover {
    background: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }

  /* AI Summary Section */
  .summary-section {
    padding: var(--space-md);
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%);
    border-bottom: 1px solid rgba(99, 102, 241, 0.3);
  }

  .summary-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-sm);
  }

  .summary-title {
    font-weight: 600;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }

  .cached-badge {
    font-size: var(--font-xs);
    opacity: 0.7;
  }

  .summary-controls {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }

  .style-select {
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-xs);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    cursor: pointer;
  }

  .style-select:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  .summary-btn {
    padding: var(--space-xs);
    font-size: var(--font-sm);
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .summary-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .summary-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .summary-loading {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    color: var(--text-muted);
    font-size: var(--font-sm);
  }

  .spinner-small {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  .summary-error {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm);
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: var(--radius-sm);
    color: #f87171;
    font-size: var(--font-sm);
  }

  .retry-btn {
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-xs);
    background: rgba(239, 68, 68, 0.2);
    border: 1px solid rgba(239, 68, 68, 0.4);
    border-radius: var(--radius-sm);
    color: #f87171;
    cursor: pointer;
    margin-left: auto;
  }

  .retry-btn:hover {
    background: rgba(239, 68, 68, 0.3);
  }

  .summary-content {
    font-size: var(--font-sm);
    line-height: 1.6;
    color: var(--text-primary);
  }

  .summary-content p {
    margin: 0;
    white-space: pre-wrap;
  }

  .key-points {
    margin-top: var(--space-sm);
    padding-top: var(--space-sm);
    border-top: 1px solid rgba(99, 102, 241, 0.2);
  }

  .key-points strong {
    display: block;
    margin-bottom: var(--space-xs);
    color: var(--text-secondary);
    font-size: var(--font-xs);
  }

  .key-points ul {
    margin: 0;
    padding-left: var(--space-md);
  }

  .key-points li {
    margin-bottom: var(--space-xs);
    color: var(--text-secondary);
  }

  .action.active {
    background: var(--accent-primary);
    color: white;
  }
</style>
