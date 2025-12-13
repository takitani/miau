<script>
  import { onMount } from 'svelte';
  import DOMPurify from 'dompurify';
  import { archiveEmail, deleteEmail, toggleStar, markAsRead } from '../stores/emails.js';
  import { showCompose } from '../stores/ui.js';

  export let email;

  // Configure DOMPurify for email-safe HTML
  const DOMPURIFY_CONFIG = {
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
    ALLOWED_ATTR: [
      'align', 'alt', 'bgcolor', 'border', 'cellpadding', 'cellspacing',
      'class', 'color', 'cols', 'colspan', 'coords', 'dir', 'disabled',
      'height', 'href', 'hspace', 'id', 'lang', 'name', 'noshade', 'nowrap',
      'rel', 'rows', 'rowspan', 'rules', 'scope', 'shape', 'size',
      'span', 'src', 'start', 'style', 'summary', 'tabindex', 'target',
      'title', 'type', 'usemap', 'valign', 'value', 'vspace', 'width',
      'data-blocked-src'
    ],
    FORBID_TAGS: ['script', 'style', 'meta', 'link', 'base', 'object', 'embed', 'applet', 'frame', 'frameset', 'iframe'],
    FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'],
    ALLOW_DATA_ATTR: true,
    ADD_ATTR: ['target'],
  };

  // Hook to force external links to open in new tab
  DOMPurify.addHook('afterSanitizeAttributes', function(node) {
    if (node.tagName === 'A' && node.hasAttribute('href')) {
      const href = node.getAttribute('href');
      if (href && (href.startsWith('http://') || href.startsWith('https://'))) {
        node.setAttribute('target', '_blank');
        node.setAttribute('rel', 'noopener noreferrer');
      }
    }
  });

  // State
  let fullEmail = null;
  let loading = false;
  let showImages = false;
  let hasExternalImages = false;
  let processedHtml = '';
  let showDetails = false;

  // AI Summary state
  let summary = null;
  let summaryLoading = false;
  let summaryError = null;
  let showSummary = false;
  let summaryStyle = 'brief';

  // Load full email when email changes
  $: if (email?.id) {
    loadFullEmail(email.id);
    summary = null;
    summaryError = null;
    showSummary = false;
    loadCachedSummary(email.id);
  }

  async function loadFullEmail(id) {
    loading = true;
    showImages = false;
    hasExternalImages = false;
    fullEmail = null;
    try {
      if (window.go?.desktop?.App) {
        fullEmail = await window.go.desktop.App.GetEmail(id);
      } else {
        fullEmail = {
          ...email,
          toAddresses: 'me@example.com',
          ccAddresses: '',
          bodyText: 'This is the email body content.\n\nBest regards,\nSender',
          bodyHtml: '<div><p>Test HTML</p></div>'
        };
      }
    } catch (err) {
      console.error('Failed to load email:', err);
    } finally {
      loading = false;
    }
  }

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
      // Cache miss is expected
    }
  }

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
    } finally {
      summaryLoading = false;
    }
  }

  async function changeSummaryStyle(newStyle) {
    summaryStyle = newStyle;
    await generateSummary();
  }

  function toggleSummaryPanel() {
    if (!showSummary && !summary) {
      generateSummary();
    } else {
      showSummary = !showSummary;
    }
  }

  async function refreshSummary() {
    if (!email?.id) return;
    try {
      if (window.go?.desktop?.App) {
        await window.go.desktop.App.InvalidateSummary(email.id);
      }
    } catch (err) {}
    await generateSummary();
  }

  // Process HTML for display
  $: if (fullEmail?.bodyHtml) {
    processHtml(fullEmail.bodyHtml);
  }

  function processHtml(html) {
    if (!html) {
      processedHtml = '';
      return;
    }

    hasExternalImages = /src=["']https?:\/\//i.test(html);
    let processed = html;

    // Convert cid: URLs to data: URLs
    if (fullEmail?.attachments) {
      fullEmail.attachments.forEach(att => {
        if (att.isInline && att.contentId && att.data) {
          const cidPattern = new RegExp(`src=["']cid:${att.contentId.replace(/[<>]/g, '')}["']`, 'gi');
          const dataUrl = `src="data:${att.contentType};base64,${att.data}"`;
          processed = processed.replace(cidPattern, dataUrl);
        }
      });
    }

    // Block external images
    if (!showImages && hasExternalImages) {
      processed = processed.replace(
        /<img([^>]*)src=["'](https?:\/\/[^"']+)["']([^>]*)>/gi,
        '<img$1src="data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'120\' height=\'60\'%3E%3Crect fill=\'%233c3c3c\' width=\'120\' height=\'60\' rx=\'4\'/%3E%3Ctext x=\'60\' y=\'35\' fill=\'%239aa0a6\' text-anchor=\'middle\' font-size=\'11\'%3EImage blocked%3C/text%3E%3C/svg%3E" data-blocked-src="$2"$3 class="blocked-image" title="External image blocked">'
      );
    }

    processedHtml = DOMPurify.sanitize(processed, DOMPURIFY_CONFIG);
  }

  function loadExternalImages() {
    showImages = true;
    if (fullEmail?.bodyHtml) {
      processHtml(fullEmail.bodyHtml);
    }
  }

  // Format date - relative for recent, full for older
  function formatDate(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) {
      return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
    } else if (days === 1) {
      return 'Ontem, ' + date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
    } else if (days < 7) {
      return date.toLocaleDateString('pt-BR', { weekday: 'long', hour: '2-digit', minute: '2-digit' });
    } else {
      return date.toLocaleDateString('pt-BR', { day: 'numeric', month: 'short', year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined });
    }
  }

  function formatFullDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString('pt-BR', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  // Get sender initials for avatar
  function getInitials(name, email) {
    if (name) {
      const parts = name.split(' ');
      if (parts.length >= 2) {
        return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
      }
      return name.substring(0, 2).toUpperCase();
    }
    return email?.substring(0, 2).toUpperCase() || '?';
  }

  // Actions
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
    window.composeContext = { mode: 'reply', replyTo: fullEmail || email };
    showCompose.set(true);
  }

  function handleReplyAll() {
    window.composeContext = { mode: 'replyAll', replyTo: fullEmail || email };
    showCompose.set(true);
  }

  function handleForward() {
    window.composeContext = { mode: 'forward', forwardEmail: fullEmail || email };
    showCompose.set(true);
  }

  // Attachment helpers
  function getAttachmentIcon(contentType) {
    if (!contentType) return 'file';
    if (contentType.startsWith('image/')) return 'image';
    if (contentType.startsWith('video/')) return 'video';
    if (contentType.startsWith('audio/')) return 'audio';
    if (contentType.includes('pdf')) return 'pdf';
    if (contentType.includes('word') || contentType.includes('document')) return 'doc';
    if (contentType.includes('excel') || contentType.includes('sheet')) return 'sheet';
    if (contentType.includes('zip') || contentType.includes('compressed')) return 'zip';
    return 'file';
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return Math.round(bytes / 1024) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  async function downloadAttachment(att) {
    try {
      if (window.go?.desktop?.App) {
        const result = await window.go.desktop.App.SaveAttachmentByPart(email.id, att.partNumber, att.filename);
        if (result) console.log('Saved to:', result);
      }
    } catch (err) {
      console.error('Failed to download:', err);
      alert('Erro ao baixar anexo: ' + err.message);
    }
  }

  async function openAttachment(att) {
    try {
      if (window.go?.desktop?.App) {
        await window.go.desktop.App.OpenAttachmentByPart(email.id, att.partNumber, att.filename);
      }
    } catch (err) {
      console.error('Failed to open:', err);
      alert('Erro ao abrir anexo: ' + err.message);
    }
  }

  // Non-inline attachments only
  $: regularAttachments = fullEmail?.attachments?.filter(a => !a.isInline) || [];
</script>

<div class="email-viewer">
  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <span>Carregando email...</span>
    </div>
  {:else if email}
    <!-- Compact Toolbar -->
    <div class="toolbar">
      <div class="toolbar-left">
        <button class="icon-btn" title="Arquivar (e)" on:click={handleArchive}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 8v13H3V8M1 3h22v5H1zM10 12h4"/>
          </svg>
        </button>
        <button class="icon-btn" title="Excluir (x)" on:click={handleDelete}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14"/>
          </svg>
        </button>
        <button class="icon-btn" title="Marcar como não lido (u)" on:click={handleMarkUnread}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 12h-6l-2 3h-4l-2-3H2"/>
            <path d="M5.45 5.11L2 12v6a2 2 0 002 2h16a2 2 0 002-2v-6l-3.45-6.89A2 2 0 0016.76 4H7.24a2 2 0 00-1.79 1.11z"/>
          </svg>
        </button>
        <div class="toolbar-divider"></div>
        <button
          class="icon-btn"
          class:active={showSummary}
          title="Resumo IA (s)"
          on:click={toggleSummaryPanel}
          disabled={summaryLoading}
        >
          {#if summaryLoading}
            <div class="spinner-small"></div>
          {:else}
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3M12 17h.01"/>
            </svg>
          {/if}
        </button>
      </div>
      <div class="toolbar-right">
        <button class="icon-btn" class:starred={email.isStarred} title="Estrela (s)" on:click={handleStar}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill={email.isStarred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Email Content Area -->
    <div class="email-content">
      <!-- Subject -->
      <h1 class="subject">{email.subject || '(sem assunto)'}</h1>

      <!-- Sender Card -->
      <div class="sender-card">
        <div class="avatar" style="background: {email.fromName ? '#' + Math.abs(email.fromName.charCodeAt(0) * 123456).toString(16).slice(0,6) : 'var(--avatar-bg)'}">
          {getInitials(email.fromName, email.fromEmail)}
        </div>
        <div class="sender-info">
          <div class="sender-row">
            <span class="sender-name">{email.fromName || email.fromEmail}</span>
            <span class="date">{formatDate(email.date)}</span>
          </div>
          <button class="recipients-toggle" on:click={() => showDetails = !showDetails}>
            <span class="to-label">para mim</span>
            <svg class="chevron" class:open={showDetails} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </button>
        </div>
        <div class="sender-actions">
          <button class="reply-btn" on:click={handleReply}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 17l-5-5 5-5M4 12h16"/>
            </svg>
          </button>
          <button class="more-btn">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <circle cx="12" cy="5" r="1.5"/>
              <circle cx="12" cy="12" r="1.5"/>
              <circle cx="12" cy="19" r="1.5"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Details Panel -->
      {#if showDetails}
        <div class="details-panel">
          <div class="detail-row">
            <span class="detail-label">de:</span>
            <span class="detail-value">{email.fromName} &lt;{email.fromEmail}&gt;</span>
          </div>
          {#if fullEmail?.toAddresses}
            <div class="detail-row">
              <span class="detail-label">para:</span>
              <span class="detail-value">{fullEmail.toAddresses}</span>
            </div>
          {/if}
          {#if fullEmail?.ccAddresses}
            <div class="detail-row">
              <span class="detail-label">cc:</span>
              <span class="detail-value">{fullEmail.ccAddresses}</span>
            </div>
          {/if}
          <div class="detail-row">
            <span class="detail-label">data:</span>
            <span class="detail-value">{formatFullDate(email.date)}</span>
          </div>
        </div>
      {/if}

      <!-- Image Warning -->
      {#if hasExternalImages && !showImages}
        <div class="image-warning">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
          <span>Imagens externas bloqueadas</span>
          <button on:click={loadExternalImages}>Mostrar imagens</button>
        </div>
      {/if}

      <!-- AI Summary -->
      {#if showSummary || summaryLoading || summaryError}
        <div class="ai-summary">
          <div class="summary-header">
            <div class="summary-title">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3M12 17h.01"/>
              </svg>
              Resumo IA
              {#if summary?.cached}
                <span class="cached-tag">cached</span>
              {/if}
            </div>
            <div class="summary-actions">
              <select
                class="style-select"
                bind:value={summaryStyle}
                on:change={() => changeSummaryStyle(summaryStyle)}
                disabled={summaryLoading}
              >
                <option value="tldr">TL;DR</option>
                <option value="brief">Breve</option>
                <option value="detailed">Detalhado</option>
              </select>
              <button class="icon-btn small" title="Atualizar" on:click={refreshSummary} disabled={summaryLoading}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M23 4v6h-6M1 20v-6h6"/>
                  <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
                </svg>
              </button>
              <button class="icon-btn small" title="Fechar" on:click={() => showSummary = false}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 6L6 18M6 6l12 12"/>
                </svg>
              </button>
            </div>
          </div>

          {#if summaryLoading}
            <div class="summary-loading">
              <div class="spinner-small"></div>
              <span>Gerando resumo...</span>
            </div>
          {:else if summaryError}
            <div class="summary-error">
              <span>{summaryError}</span>
              <button on:click={generateSummary}>Tentar novamente</button>
            </div>
          {:else if summary}
            <div class="summary-body">
              <p>{summary.content}</p>
              {#if summary.keyPoints?.length > 0}
                <ul class="key-points">
                  {#each summary.keyPoints as point}
                    <li>{point}</li>
                  {/each}
                </ul>
              {/if}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Email Body -->
      <div class="email-body">
        {#if fullEmail?.bodyHtml}
          <div class="html-content">
            {@html processedHtml}
          </div>
        {:else if fullEmail?.bodyText}
          <pre class="text-content">{fullEmail.bodyText}</pre>
        {:else}
          <p class="snippet">{email.snippet}</p>
        {/if}
      </div>

      <!-- Attachments -->
      {#if regularAttachments.length > 0}
        <div class="attachments-section">
          <div class="attachments-header">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/>
            </svg>
            <span>{regularAttachments.length} anexo{regularAttachments.length !== 1 ? 's' : ''}</span>
          </div>
          <div class="attachments-grid">
            {#each regularAttachments as att}
              <div class="attachment-card">
                <div class="attachment-icon" class:image={getAttachmentIcon(att.contentType) === 'image'}>
                  {#if getAttachmentIcon(att.contentType) === 'image'}
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                      <circle cx="8.5" cy="8.5" r="1.5"/>
                      <polyline points="21 15 16 10 5 21"/>
                    </svg>
                  {:else if getAttachmentIcon(att.contentType) === 'pdf'}
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
                      <polyline points="14 2 14 8 20 8"/>
                      <line x1="16" y1="13" x2="8" y2="13"/>
                      <line x1="16" y1="17" x2="8" y2="17"/>
                    </svg>
                  {:else}
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z"/>
                      <polyline points="13 2 13 9 20 9"/>
                    </svg>
                  {/if}
                </div>
                <div class="attachment-info">
                  <span class="attachment-name">{att.filename}</span>
                  <span class="attachment-size">{formatSize(att.size)}</span>
                </div>
                <div class="attachment-actions">
                  <button class="icon-btn small" title="Abrir" on:click={() => openAttachment(att)}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6"/>
                      <polyline points="15 3 21 3 21 9"/>
                      <line x1="10" y1="14" x2="21" y2="3"/>
                    </svg>
                  </button>
                  <button class="icon-btn small" title="Baixar" on:click={() => downloadAttachment(att)}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                      <polyline points="7 10 12 15 17 10"/>
                      <line x1="12" y1="15" x2="12" y2="3"/>
                    </svg>
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Quick Reply Actions -->
      <div class="reply-actions">
        <button class="reply-action" on:click={handleReply}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 17l-5-5 5-5M4 12h16"/>
          </svg>
          Responder
        </button>
        <button class="reply-action" on:click={handleReplyAll}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 17l-5-5 5-5M4 12h16"/>
            <path d="M13 17l-5-5 5-5" opacity="0.5"/>
          </svg>
          Responder a todos
        </button>
        <button class="reply-action" on:click={handleForward}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 17l5-5-5-5M20 12H4"/>
          </svg>
          Encaminhar
        </button>
      </div>
    </div>
  {:else}
    <div class="empty-state">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <path d="M22 12h-6l-2 3h-4l-2-3H2"/>
        <path d="M5.45 5.11L2 12v6a2 2 0 002 2h16a2 2 0 002-2v-6l-3.45-6.89A2 2 0 0016.76 4H7.24a2 2 0 00-1.79 1.11z"/>
      </svg>
      <p>Selecione um email para visualizar</p>
    </div>
  {/if}
</div>

<style>
  .email-viewer {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
  }

  /* Loading State */
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: var(--space-md);
    color: var(--text-muted);
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .spinner-small {
    width: 16px;
    height: 16px;
    border: 2px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Toolbar */
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-xs) var(--space-sm);
    border-bottom: 1px solid var(--border-subtle);
    background: var(--bg-secondary);
  }

  .toolbar-left, .toolbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }

  .toolbar-divider {
    width: 1px;
    height: 24px;
    background: var(--border-color);
    margin: 0 var(--space-xs);
  }

  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border-radius: var(--radius-full);
    color: var(--text-secondary);
    transition: background var(--transition-fast), color var(--transition-fast);
  }

  .icon-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .icon-btn.active {
    background: var(--bg-selected);
    color: var(--accent-primary);
  }

  .icon-btn.starred {
    color: var(--accent-warning);
  }

  .icon-btn.small {
    width: 28px;
    height: 28px;
  }

  .icon-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Email Content */
  .email-content {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-lg) var(--space-xl);
  }

  .subject {
    font-size: var(--font-2xl);
    font-weight: var(--weight-normal);
    color: var(--text-primary);
    line-height: 1.3;
    margin-bottom: var(--space-lg);
  }

  /* Sender Card */
  .sender-card {
    display: flex;
    align-items: flex-start;
    gap: var(--space-md);
    margin-bottom: var(--space-md);
  }

  .avatar {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-full);
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: var(--font-sm);
    font-weight: var(--weight-medium);
    flex-shrink: 0;
  }

  .sender-info {
    flex: 1;
    min-width: 0;
  }

  .sender-row {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .sender-name {
    font-size: var(--font-md);
    font-weight: var(--weight-medium);
    color: var(--text-primary);
  }

  .date {
    font-size: var(--font-sm);
    color: var(--text-muted);
  }

  .recipients-toggle {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    padding: 2px 0;
    font-size: var(--font-sm);
    color: var(--text-muted);
    background: none;
    border: none;
    cursor: pointer;
  }

  .recipients-toggle:hover {
    color: var(--text-secondary);
  }

  .chevron {
    transition: transform var(--transition-fast);
  }

  .chevron.open {
    transform: rotate(180deg);
  }

  .sender-actions {
    display: flex;
    gap: var(--space-xs);
  }

  .reply-btn, .more-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: var(--radius-full);
    color: var(--text-secondary);
    transition: background var(--transition-fast);
  }

  .reply-btn:hover, .more-btn:hover {
    background: var(--bg-hover);
  }

  /* Details Panel */
  .details-panel {
    margin-left: 56px;
    margin-bottom: var(--space-md);
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-secondary);
    border-radius: var(--radius-md);
    font-size: var(--font-sm);
  }

  .detail-row {
    display: flex;
    gap: var(--space-sm);
    padding: var(--space-xs) 0;
  }

  .detail-label {
    color: var(--text-muted);
    min-width: 40px;
  }

  .detail-value {
    color: var(--text-secondary);
    word-break: break-all;
  }

  /* Image Warning */
  .image-warning {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-secondary);
    border-radius: var(--radius-md);
    margin-bottom: var(--space-md);
    font-size: var(--font-sm);
    color: var(--text-secondary);
  }

  .image-warning button {
    margin-left: auto;
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    font-size: var(--font-xs);
    color: var(--accent-primary);
  }

  .image-warning button:hover {
    background: var(--bg-hover);
  }

  /* AI Summary */
  .ai-summary {
    margin-bottom: var(--space-md);
    padding: var(--space-md);
    background: linear-gradient(135deg, rgba(26, 115, 232, 0.08) 0%, rgba(156, 39, 176, 0.08) 100%);
    border: 1px solid rgba(26, 115, 232, 0.2);
    border-radius: var(--radius-md);
  }

  .summary-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-sm);
  }

  .summary-title {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: var(--font-sm);
    font-weight: var(--weight-medium);
    color: var(--accent-primary);
  }

  .cached-tag {
    font-size: var(--font-xs);
    padding: 1px 6px;
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    color: var(--text-muted);
  }

  .summary-actions {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }

  .style-select {
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-xs);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
  }

  .summary-loading {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    font-size: var(--font-sm);
    color: var(--text-muted);
  }

  .summary-error {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    font-size: var(--font-sm);
    color: var(--accent-error);
  }

  .summary-error button {
    padding: var(--space-xs) var(--space-sm);
    background: rgba(234, 67, 53, 0.1);
    border-radius: var(--radius-sm);
    font-size: var(--font-xs);
    color: var(--accent-error);
  }

  .summary-body {
    font-size: var(--font-sm);
    line-height: 1.6;
    color: var(--text-primary);
  }

  .summary-body p {
    margin: 0;
  }

  .key-points {
    margin-top: var(--space-sm);
    padding-left: var(--space-md);
    color: var(--text-secondary);
  }

  .key-points li {
    margin-bottom: var(--space-xs);
  }

  /* Email Body */
  .email-body {
    margin-bottom: var(--space-lg);
    border: none;
  }

  .html-content {
    background: transparent;
    padding: 0;
    color: var(--text-primary);
    font-size: 15px;
    line-height: 1.6;
    word-wrap: break-word;
    overflow-wrap: break-word;
    border: none;
  }

  /* AGGRESSIVE reset - remove ALL borders and force text color for dark theme */
  .html-content :global(*) {
    border: 0 none transparent !important;
    border-width: 0 !important;
    border-style: none !important;
    border-color: transparent !important;
    outline: none !important;
    box-shadow: none !important;
    color: var(--text-primary) !important;
    background-color: transparent !important;
  }

  .html-content :global(a) {
    color: var(--email-link-color) !important;
  }

  .html-content :global(img) {
    max-width: 100%;
    height: auto;
  }

  .html-content :global(img.blocked-image) {
    cursor: pointer;
  }

  /* Only blockquote keeps left border for visual hierarchy */
  .html-content :global(blockquote) {
    margin: var(--space-sm) 0;
    padding-left: var(--space-md);
    border-left: 3px solid var(--email-quote-border) !important;
    color: var(--email-quote-text) !important;
  }

  .html-content :global(pre),
  .html-content :global(code) {
    background: var(--email-code-bg) !important;
    padding: 2px 6px;
    border-radius: var(--radius-sm);
    font-family: 'Roboto Mono', monospace;
    font-size: var(--font-sm);
  }

  .html-content :global(pre) {
    padding: var(--space-md);
    overflow-x: auto;
  }

  .html-content :global(table) {
    border-collapse: collapse;
    max-width: 100%;
  }

  .html-content :global(hr) {
    border: none !important;
    border-top: 1px solid var(--border-subtle) !important;
    margin: var(--space-md) 0;
  }

  .text-content {
    white-space: pre-wrap;
    font-family: inherit;
    font-size: 15px;
    line-height: 1.6;
    color: var(--text-primary);
    margin: 0;
  }

  .snippet {
    color: var(--text-secondary);
    font-style: italic;
  }

  /* Attachments */
  .attachments-section {
    margin-bottom: var(--space-lg);
  }

  .attachments-header {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    font-size: var(--font-sm);
    color: var(--text-muted);
    margin-bottom: var(--space-sm);
  }

  .attachments-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: var(--space-sm);
  }

  .attachment-card {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm);
    background: var(--bg-secondary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    transition: border-color var(--transition-fast);
  }

  .attachment-card:hover {
    border-color: var(--border-color);
  }

  .attachment-icon {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .attachment-icon.image {
    background: rgba(52, 168, 83, 0.15);
    color: var(--accent-success);
  }

  .attachment-info {
    flex: 1;
    min-width: 0;
  }

  .attachment-name {
    display: block;
    font-size: var(--font-sm);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .attachment-size {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .attachment-actions {
    display: flex;
    gap: var(--space-xs);
  }

  /* Reply Actions */
  .reply-actions {
    display: flex;
    gap: var(--space-sm);
    padding-top: var(--space-lg);
  }

  .reply-action {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: transparent;
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-full);
    font-size: var(--font-sm);
    color: var(--text-secondary);
    transition: all var(--transition-fast);
  }

  .reply-action:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
    border-color: var(--text-muted);
  }

  /* Empty State */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: var(--space-md);
    color: var(--text-muted);
  }

  .empty-state svg {
    opacity: 0.5;
  }

  .empty-state p {
    font-size: var(--font-md);
  }
</style>
