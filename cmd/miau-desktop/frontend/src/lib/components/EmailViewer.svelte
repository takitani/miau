<script>
  import { archiveEmail, deleteEmail, toggleStar, markAsRead } from '../stores/emails.js';

  export let email;

  // Full email content (loaded on demand)
  let fullEmail = null;
  let loading = false;

  // Load full email when email changes
  $: if (email && email.id) {
    loadFullEmail(email.id);
  }

  async function loadFullEmail(id) {
    loading = true;
    try {
      if (window.go?.desktop?.App) {
        fullEmail = await window.go.desktop.App.GetEmail(id);
      } else {
        // Mock for development
        fullEmail = {
          ...email,
          toAddresses: 'me@example.com',
          ccAddresses: '',
          bodyText: 'This is the email body content.\n\nBest regards,\nSender',
          bodyHtml: ''
        };
      }
    } catch (err) {
      console.error('Failed to load email:', err);
    } finally {
      loading = false;
    }
  }

  // Format date
  function formatDate(dateStr) {
    const date = new Date(dateStr);
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
      <button class="action" title="Arquivar (e)" on:click={handleArchive}>
        📁 Arquivar
      </button>
      <button class="action" title="Excluir (x)" on:click={handleDelete}>
        🗑️ Excluir
      </button>
      <button class="action" title="Estrela (s)" on:click={handleStar}>
        {email.isStarred ? '★' : '☆'} Estrela
      </button>
      <button class="action" title="Marcar não lido (u)" on:click={handleMarkUnread}>
        ✉️ Não lido
      </button>
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

    <!-- Body -->
    <div class="email-body">
      {#if fullEmail?.bodyHtml}
        <!-- Render HTML (sanitized) -->
        {@html fullEmail.bodyHtml}
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
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-secondary);
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
    padding: var(--space-lg);
    overflow-y: auto;
  }

  .text-body {
    white-space: pre-wrap;
    font-family: inherit;
    font-size: var(--font-md);
    line-height: 1.6;
    color: var(--text-primary);
  }

  .snippet {
    color: var(--text-secondary);
    font-style: italic;
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
