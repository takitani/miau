<script>
  import { onMount } from 'svelte';
  import { archiveEmail, deleteEmail, toggleStar, markAsRead } from '../stores/emails.js';
  import { showCompose } from '../stores/ui.js';

  export let email;

  // Full email content (loaded on demand)
  let fullEmail = null;
  let loading = false;
  let showImages = false; // User must opt-in to load external images
  let hasExternalImages = false;

  // Load full email when email changes
  $: if (email && email.id) {
    loadFullEmail(email.id);
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

    // Debug: log image sources found in HTML
    var imgMatches = html.match(/<img[^>]*src=["']([^"']+)["'][^>]*>/gi) || [];
    console.log('[EmailViewer] Found images:', imgMatches.length);
    imgMatches.forEach((img, i) => {
      var srcMatch = img.match(/src=["']([^"']+)["']/i);
      if (srcMatch) {
        var src = srcMatch[1];
        var type = src.startsWith('data:') ? 'data:' : src.startsWith('cid:') ? 'cid:' : src.startsWith('http') ? 'http' : 'other';
        console.log(`[EmailViewer] Image ${i}: type=${type}, src=${src.substring(0, 100)}...`);
      }
    });

    // Check if there are external images (http/https URLs, not data: or cid:)
    hasExternalImages = /src=["']https?:\/\//i.test(html);

    // Strip scripts for security
    var safe = html.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '');

    // Convert cid: URLs to data: URLs using inline attachments
    if (fullEmail?.attachments) {
      fullEmail.attachments.forEach(att => {
        if (att.isInline && att.contentId && att.data) {
          var cidPattern = new RegExp(`src=["']cid:${att.contentId.replace(/[<>]/g, '')}["']`, 'gi');
          var dataUrl = `src="data:${att.contentType};base64,${att.data}"`;
          safe = safe.replace(cidPattern, dataUrl);
          console.log(`[EmailViewer] Replaced cid:${att.contentId} with data URL`);
        }
      });
    }

    // Block external images unless user opts in
    if (!showImages && hasExternalImages) {
      safe = safe.replace(
        /<img([^>]*)src=["'](https?:\/\/[^"']+)["']([^>]*)>/gi,
        '<img$1src="data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'120\' height=\'80\'%3E%3Crect fill=\'%232a2a3e\' width=\'120\' height=\'80\' rx=\'4\'/%3E%3Ctext x=\'60\' y=\'45\' fill=\'%23888\' text-anchor=\'middle\' font-size=\'11\'%3EImagem bloqueada%3C/text%3E%3C/svg%3E" data-blocked-src="$2"$3 style="cursor:pointer;border:1px dashed #444;" title="Imagem externa bloqueada">'
      );
    }

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
        <h3>Anexos ({fullEmail.attachments.length})</h3>
        <ul>
          {#each fullEmail.attachments as att}
            <li class="attachment">
              <span class="icon">📎</span>
              <span class="name">{att.filename}</span>
              <span class="size">({Math.round(att.size / 1024)} KB)</span>
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
    gap: var(--space-xs);
    padding: var(--space-xs);
    font-size: var(--font-sm);
  }

  .attachment .name {
    color: var(--accent-primary);
  }

  .attachment .size {
    color: var(--text-muted);
  }
</style>
