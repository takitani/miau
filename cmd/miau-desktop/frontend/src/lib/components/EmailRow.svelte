<script>
  import { createEventDispatcher } from 'svelte';

  export let email;
  export let selected = false;

  const dispatch = createEventDispatcher();

  // Format date
  function formatDate(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;

    // Today: show time
    if (diff < 86400000 && date.getDate() === now.getDate()) {
      return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
    }

    // This year: show day/month
    if (date.getFullYear() === now.getFullYear()) {
      return date.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' });
    }

    // Other: show full date
    return date.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: '2-digit' });
  }

  // Handle click
  function handleClick() {
    dispatch('click');
  }

  // Handle double click to open
  function handleDoubleClick() {
    dispatch('open');
  }
</script>

<div
  class="email-row"
  class:selected
  class:unread={!email.isRead}
  role="button"
  tabindex="0"
  on:click={handleClick}
  on:dblclick={handleDoubleClick}
  on:keydown={(e) => e.key === 'Enter' && handleClick()}
>
  <div class="flags">
    {#if email.isStarred}
      <span class="star" title="Starred">★</span>
    {:else}
      <span class="star empty">☆</span>
    {/if}
  </div>

  <div class="from truncate">
    {email.fromName || email.fromEmail}
  </div>

  <div class="content">
    <span class="subject truncate">{email.subject || '(sem assunto)'}</span>
    <span class="separator"> - </span>
    <span class="snippet truncate">{email.snippet}</span>
  </div>

  <div class="meta">
    {#if email.hasAttachments}
      <span class="attachment" title="Has attachments">📎</span>
    {/if}
    <span class="date">{formatDate(email.date)}</span>
  </div>
</div>

<style>
  .email-row {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    border-bottom: 1px solid var(--border-color);
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .email-row:hover {
    background: var(--bg-hover);
  }

  .email-row.selected {
    background: var(--bg-selected);
  }

  .email-row.unread {
    font-weight: 600;
  }

  .email-row.unread .from,
  .email-row.unread .subject {
    color: var(--text-primary);
  }

  .flags {
    flex-shrink: 0;
    width: 20px;
    text-align: center;
  }

  .star {
    color: var(--accent-warning);
    font-size: var(--font-md);
  }

  .star.empty {
    color: var(--text-muted);
    opacity: 0.3;
  }

  .from {
    flex-shrink: 0;
    width: 180px;
    color: var(--text-secondary);
    font-size: var(--font-sm);
  }

  .content {
    flex: 1;
    display: flex;
    align-items: center;
    min-width: 0;
    overflow: hidden;
  }

  .subject {
    color: var(--text-primary);
    font-size: var(--font-sm);
  }

  .separator {
    color: var(--text-muted);
    flex-shrink: 0;
    margin: 0 var(--space-xs);
  }

  .snippet {
    color: var(--text-muted);
    font-size: var(--font-sm);
    font-weight: 400;
  }

  .meta {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    flex-shrink: 0;
  }

  .attachment {
    font-size: var(--font-sm);
  }

  .date {
    color: var(--text-muted);
    font-size: var(--font-xs);
    min-width: 60px;
    text-align: right;
  }
</style>
