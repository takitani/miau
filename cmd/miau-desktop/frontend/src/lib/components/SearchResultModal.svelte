<script>
  import { onMount } from 'svelte';
  import DOMPurify from 'dompurify';
  import { showSearchResultModal, searchResultEmailId, closeSearchResultModal } from '../stores/ui.js';

  // Configure DOMPurify
  const DOMPURIFY_CONFIG = {
    ALLOWED_TAGS: [
      'a', 'abbr', 'b', 'blockquote', 'br', 'code', 'div', 'em',
      'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'i', 'img', 'li',
      'ol', 'p', 'pre', 's', 'span', 'strong', 'sub', 'sup',
      'table', 'tbody', 'td', 'tfoot', 'th', 'thead', 'tr', 'u', 'ul'
    ],
    ALLOWED_ATTR: [
      'align', 'alt', 'bgcolor', 'border', 'class', 'color', 'height',
      'href', 'id', 'src', 'style', 'target', 'title', 'width'
    ],
    FORBID_TAGS: ['script', 'style', 'meta', 'link', 'iframe'],
    FORBID_ATTR: ['onerror', 'onload', 'onclick'],
  };

  let email = null;
  let loading = false;

  // Load email when ID changes
  $: if ($searchResultEmailId) {
    loadEmail($searchResultEmailId);
  }

  async function loadEmail(id) {
    loading = true;
    try {
      if (window.go?.desktop?.App) {
        email = await window.go.desktop.App.GetEmail(id);
      }
    } catch (err) {
      console.error('Failed to load email:', err);
    } finally {
      loading = false;
    }
  }

  function close() {
    closeSearchResultModal();
    email = null;
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    }
  }

  function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleDateString('pt-BR', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getSafeHtml(html) {
    if (!html) return '';
    return DOMPurify.sanitize(html, DOMPURIFY_CONFIG);
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if $showSearchResultModal}
  <div class="modal-overlay" on:click|self={close}>
    <div class="modal">
      <div class="modal-header">
        <h2>Email da Busca</h2>
        <button class="close-btn" on:click={close}>✕</button>
      </div>

      {#if loading}
        <div class="loading">Carregando...</div>
      {:else if email}
        <div class="email-header">
          <div class="subject">{email.subject || '(sem assunto)'}</div>
          <div class="meta">
            <span class="from">De: {email.fromName || email.fromEmail}</span>
            <span class="date">{formatDate(email.date)}</span>
          </div>
          {#if email.toAddresses}
            <div class="to">Para: {email.toAddresses}</div>
          {/if}
        </div>

        <div class="email-body">
          {#if email.bodyHtml}
            {@html getSafeHtml(email.bodyHtml)}
          {:else if email.bodyText}
            <pre class="text-body">{email.bodyText}</pre>
          {:else}
            <p class="no-content">(sem conteudo)</p>
          {/if}
        </div>
      {:else}
        <div class="error">Email nao encontrado</div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1001;
  }

  .modal {
    width: 90%;
    max-width: 800px;
    max-height: 85vh;
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-lg);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-tertiary);
  }

  .modal-header h2 {
    margin: 0;
    font-size: var(--font-md);
    color: var(--text-muted);
  }

  .close-btn {
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-lg);
    color: var(--text-muted);
    background: transparent;
    border: none;
    cursor: pointer;
    border-radius: var(--radius-sm);
  }

  .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .loading, .error {
    padding: var(--space-xl);
    text-align: center;
    color: var(--text-muted);
  }

  .email-header {
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-color);
  }

  .subject {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: var(--space-sm);
  }

  .meta {
    display: flex;
    justify-content: space-between;
    font-size: var(--font-sm);
    color: var(--text-secondary);
  }

  .to {
    margin-top: var(--space-xs);
    font-size: var(--font-sm);
    color: var(--text-muted);
  }

  .email-body {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-md);
    color: var(--text-primary);
    line-height: 1.6;
  }

  .text-body {
    white-space: pre-wrap;
    font-family: inherit;
    margin: 0;
  }

  .no-content {
    color: var(--text-muted);
    font-style: italic;
  }

  /* Email content styles */
  .email-body :global(a) {
    color: var(--accent-color);
  }

  .email-body :global(img) {
    max-width: 100%;
    height: auto;
  }

  .email-body :global(blockquote) {
    border-left: 3px solid var(--border-color);
    margin: var(--space-sm) 0;
    padding-left: var(--space-md);
    color: var(--text-secondary);
  }
</style>
