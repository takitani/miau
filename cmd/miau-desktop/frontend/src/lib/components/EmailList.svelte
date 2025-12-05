<script>
  import { emails, selectedIndex, selectEmail, loading, threadingEnabled, toggleThreading } from '../stores/emails.js';
  import { currentFolder } from '../stores/emails.js';
  import { showThreadView } from '../stores/ui.js';
  import EmailRow from './EmailRow.svelte';

  // Format folder name for display
  function formatFolderName(name) {
    return name.replace('[Gmail]/', '');
  }
</script>

<div class="email-list">
  <header class="list-header">
    <h2>{formatFolderName($currentFolder)}</h2>
    {#if !$showThreadView}
      <div class="header-right">
        <button
          class="thread-toggle"
          class:active={$threadingEnabled}
          on:click={toggleThreading}
          title="Toggle threading (g)"
        >
          {$threadingEnabled ? '◧ Threads' : '◨ List'}
        </button>
        <span class="count">{$emails.length} {$threadingEnabled ? 'threads' : 'emails'}</span>
      </div>
    {/if}
  </header>

  {#if $loading}
    <div class="loading">
      <span class="spinner"></span>
      Carregando...
    </div>
  {:else if $emails.length === 0}
    <div class="empty">
      <p>Nenhum email nesta pasta</p>
    </div>
  {:else}
    <div class="list-content">
      {#each $emails as email, index (email.id)}
        <EmailRow
          {email}
          selected={index === $selectedIndex}
          on:click={() => selectEmail(email.id)}
        />
      {/each}
    </div>
  {/if}
</div>

<style>
  .email-list {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-secondary);
  }

  .list-header h2 {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .thread-toggle {
    padding: 4px 8px;
    font-size: var(--font-xs);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .thread-toggle:hover {
    background: var(--bg-hover);
    border-color: var(--accent-primary);
  }

  .thread-toggle.active {
    background: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }

  .count {
    font-size: var(--font-sm);
    color: var(--text-muted);
  }

  .list-content {
    flex: 1;
    overflow-y: auto;
    scrollbar-gutter: stable; /* Reserve space for scrollbar to prevent layout shift */
  }

  .loading, .empty {
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
</style>
