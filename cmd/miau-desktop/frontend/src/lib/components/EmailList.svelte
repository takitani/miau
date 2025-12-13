<script>
  import { emails, selectedIndex, loading, threadingEnabled, toggleThreading } from '../stores/emails.js';
  import { currentFolder } from '../stores/emails.js';
  import { showThreadView, selectEmailSmart } from '../stores/ui.js';
  import { someSelected, selectedCount, selectionMode } from '../stores/selection.js';
  import EmailRow from './EmailRow.svelte';
  import SmartSelectMenu from './SmartSelectMenu.svelte';

  // Format folder name for display
  function formatFolderName(name) {
    return name.replace('[Gmail]/', '');
  }

  // Smart select menu state
  var showSmartMenu = false;
  var menuAnchorX = 0;
  var menuAnchorY = 0;

  function openSmartMenu(e) {
    var rect = e.target.getBoundingClientRect();
    menuAnchorX = rect.left;
    menuAnchorY = rect.bottom + 4;
    showSmartMenu = true;
  }
</script>

<div class="email-list">
  <header class="list-header">
    <div class="header-left">
      <h2>{formatFolderName($currentFolder)}</h2>
      {#if $someSelected}
        <span class="selection-badge">{$selectedCount}</span>
      {/if}
    </div>
    {#if !$showThreadView}
      <div class="header-right">
        <!-- Smart select button -->
        <button
          class="smart-select-btn"
          class:active={showSmartMenu || $selectionMode}
          on:click={openSmartMenu}
          title="Seleção inteligente (v)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7" rx="1"/>
            <rect x="14" y="3" width="7" height="7" rx="1"/>
            <rect x="3" y="14" width="7" height="7" rx="1"/>
            <rect x="14" y="14" width="7" height="7" rx="1"/>
          </svg>
        </button>
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
          {index}
          selected={index === $selectedIndex}
          on:click={() => selectEmailSmart(email.id)}
        />
      {/each}
    </div>
  {/if}
</div>

<!-- Smart Select Menu -->
<SmartSelectMenu
  bind:show={showSmartMenu}
  anchorX={menuAnchorX}
  anchorY={menuAnchorY}
  on:close={() => showSmartMenu = false}
/>

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

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .list-header h2 {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
  }

  .selection-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 22px;
    height: 22px;
    padding: 0 6px;
    background: var(--accent-primary);
    border-radius: 11px;
    font-size: 11px;
    font-weight: 700;
    color: var(--btn-primary-text);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .smart-select-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .smart-select-btn:hover {
    background: var(--bg-hover);
    border-color: var(--accent-primary);
    color: var(--text-primary);
  }

  .smart-select-btn.active {
    background: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }

  .smart-select-btn svg {
    width: 16px;
    height: 16px;
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
