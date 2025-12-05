<script>
  import { createEventDispatcher } from 'svelte';
  import DOMPurify from 'dompurify';

  export var message;
  export var isExpanded = false;
  export var isSelected = false;
  export var participantColor = '#666';

  var dispatch = createEventDispatcher();

  // Format relative time
  function formatTimeAgo(dateStr) {
    var date = new Date(dateStr);
    var now = new Date();
    var diff = now - date;
    var mins = Math.floor(diff / 60000);
    var hours = Math.floor(diff / 3600000);
    var days = Math.floor(diff / 86400000);

    if (mins < 1) return 'agora';
    if (mins < 60) return `${mins}m`;
    if (hours < 24) return `${hours}h`;
    if (days < 7) return `${days}d`;
    return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' });
  }

  // Extract recipients display
  function formatRecipients(addresses) {
    if (!addresses) return 'todos';
    var parts = addresses.split(',');
    if (parts.length > 1) {
      return `${parts[0].trim()} +${parts.length - 1}`;
    }
    return parts[0].trim();
  }

  // Sanitize HTML content
  function sanitizeHtml(html) {
    if (!html) return '';
    return DOMPurify.sanitize(html, {
      ALLOWED_TAGS: ['p', 'br', 'b', 'i', 'u', 'strong', 'em', 'a', 'ul', 'ol', 'li', 'blockquote', 'pre', 'code', 'div', 'span', 'h1', 'h2', 'h3', 'h4', 'table', 'tr', 'td', 'th', 'thead', 'tbody'],
      ALLOWED_ATTR: ['href', 'target', 'style', 'class'],
      ALLOW_DATA_ATTR: false
    });
  }

  function toggleExpand() {
    dispatch('toggle');
  }

  function handleClick() {
    dispatch('select');
  }

  // Generate preview from available content
  function getPreview() {
    if (message.snippet) return message.snippet;
    if (message.bodyText) {
      // Clean and truncate body text for preview
      var text = message.bodyText
        .replace(/[\r\n]+/g, ' ')
        .replace(/\s+/g, ' ')
        .trim();
      return text.length > 150 ? text.slice(0, 150) + '...' : text;
    }
    if (message.bodyHtml) {
      // Strip HTML tags and get text
      var div = document.createElement('div');
      div.innerHTML = message.bodyHtml;
      var text = (div.textContent || div.innerText || '')
        .replace(/[\r\n]+/g, ' ')
        .replace(/\s+/g, ' ')
        .trim();
      return text.length > 150 ? text.slice(0, 150) + '...' : text;
    }
    return null;
  }

  $: preview = getPreview();
</script>

<article
  class="message"
  class:expanded={isExpanded}
  class:selected={isSelected}
  class:unread={!message.isRead}
  on:click={handleClick}
  role="button"
  tabindex="0"
  on:keypress={(e) => e.key === 'Enter' && handleClick()}
>
  <!-- Participant indicator bar -->
  <div class="participant-bar" style="background: {participantColor}"></div>

  <!-- Header (always visible) -->
  <header class="message-header" on:click|stopPropagation={toggleExpand}>
    <div class="header-left">
      <span class="expand-icon">{isExpanded ? '▾' : '▸'}</span>
      <span class="avatar" style="background: {participantColor}">
        {message.fromName ? message.fromName[0].toUpperCase() : '?'}
      </span>
      <div class="sender-info">
        <span class="sender-name" class:unread={!message.isRead}>{message.fromName || message.fromEmail}</span>
        <span class="recipients">→ {formatRecipients(message.toAddresses)}</span>
      </div>
    </div>
    <div class="header-right">
      {#if message.hasAttachments}
        <span class="attachment-icon" title="Anexos">📎</span>
      {/if}
      {#if message.isStarred}
        <span class="star-icon">⭐</span>
      {/if}
      <span class="time">{formatTimeAgo(message.date)}</span>
    </div>
  </header>

  <!-- Preview (collapsed state) -->
  {#if !isExpanded}
    <div class="message-preview">
      <span class="snippet">"{preview || 'Sem preview disponível'}"</span>
    </div>
  {/if}

  <!-- Content (expanded state) -->
  {#if isExpanded}
    <div class="message-content">
      <div class="content-divider"></div>
      {#if message.bodyHtml}
        <div class="html-content">
          {@html sanitizeHtml(message.bodyHtml)}
        </div>
      {:else if message.bodyText}
        <pre class="text-content">{message.bodyText}</pre>
      {:else}
        <p class="no-content">Conteúdo não disponível</p>
      {/if}

      {#if message.hasAttachments}
        <div class="attachments-indicator">
          <span class="icon">📎</span>
          <span>Anexos disponíveis</span>
        </div>
      {/if}
    </div>
  {/if}
</article>

<style>
  .message {
    position: relative;
    background: var(--bg-secondary);
    border-radius: 8px;
    margin-bottom: 8px;
    overflow: hidden;
    cursor: pointer;
    transition: all 0.15s ease;
    border: 1px solid transparent;
  }

  .message:hover {
    background: var(--bg-hover);
  }

  .message.selected {
    border-color: var(--accent);
    background: var(--bg-active);
  }

  .message.unread {
    border-left: 3px solid var(--accent);
  }

  .participant-bar {
    position: absolute;
    top: 0;
    right: 0;
    width: 4px;
    height: 100%;
    opacity: 0.8;
  }

  .message-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    padding-right: 24px;
    gap: 12px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    min-width: 0;
  }

  .expand-icon {
    font-size: 12px;
    color: var(--text-muted);
    width: 16px;
  }

  .avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 14px;
    color: white;
    flex-shrink: 0;
  }

  .sender-info {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .sender-name {
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sender-name.unread {
    font-weight: 700;
  }

  .recipients {
    font-size: 12px;
    color: var(--text-muted);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .attachment-icon,
  .star-icon {
    font-size: 14px;
  }

  .time {
    font-size: 12px;
    color: var(--text-muted);
    min-width: 40px;
    text-align: right;
  }

  .message-preview {
    padding: 0 16px 12px 58px;
  }

  .snippet {
    font-size: 13px;
    color: var(--text-muted);
    font-style: italic;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .message-content {
    padding: 0 16px 16px 58px;
    padding-right: 24px;
  }

  .content-divider {
    height: 1px;
    background: var(--border);
    margin-bottom: 16px;
  }

  .html-content {
    font-size: 14px;
    line-height: 1.6;
    color: var(--text-primary);
    overflow-x: auto;
  }

  .html-content :global(a) {
    color: var(--accent);
  }

  .html-content :global(blockquote) {
    margin: 8px 0;
    padding-left: 12px;
    border-left: 3px solid var(--border);
    color: var(--text-muted);
  }

  .html-content :global(pre),
  .html-content :global(code) {
    background: var(--bg-primary);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: monospace;
    font-size: 13px;
  }

  .text-content {
    font-size: 14px;
    line-height: 1.6;
    color: var(--text-primary);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    font-family: inherit;
  }

  .no-content {
    color: var(--text-muted);
    font-style: italic;
  }

  .attachments-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 16px;
    padding: 8px 12px;
    background: var(--bg-primary);
    border-radius: 6px;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .attachments-indicator .icon {
    font-size: 16px;
  }

  /* Expanded state animation */
  .message.expanded {
    background: var(--bg-secondary);
  }

  .message.expanded .message-header {
    border-bottom: none;
  }
</style>
