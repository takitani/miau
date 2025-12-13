<script>
  import { connected, syncing, lastSync, syncEmails, switchToTerminal, autoRefreshInterval, autoRefreshStart, autoRefreshEnabled, newEmailCount, newEmailShowUntil } from '../stores/ui.js';
  import { toggleDebug } from '../stores/debug.js';
  import { onMount, onDestroy } from 'svelte';
  import ThemeToggle from './ThemeToggle.svelte';

  // Timer progress state
  let timerProgress = 0;
  let remainingSeconds = autoRefreshInterval;
  let updateInterval;
  let showNewEmailBadge = false;

  // Update timer progress every second
  function updateTimer() {
    if (!$autoRefreshEnabled) {
      timerProgress = 0;
      remainingSeconds = autoRefreshInterval;
      return;
    }
    const elapsed = (Date.now() - $autoRefreshStart) / 1000;
    timerProgress = Math.min(elapsed / autoRefreshInterval, 1);
    remainingSeconds = Math.max(0, Math.floor(autoRefreshInterval - elapsed));

    // Check if new email badge should be shown
    showNewEmailBadge = $newEmailCount > 0 && Date.now() < $newEmailShowUntil;
  }

  onMount(() => {
    updateInterval = setInterval(updateTimer, 200);
  });

  onDestroy(() => {
    if (updateInterval) clearInterval(updateInterval);
  });

  // Format last sync time
  function formatLastSync(date) {
    if (!date) return 'Nunca';
    const diff = Date.now() - date.getTime();

    if (diff < 60000) return 'Agora';
    if (diff < 3600000) return `${Math.floor(diff / 60000)} min atrás`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h atrás`;
    return date.toLocaleDateString('pt-BR');
  }

  // Generate progress bar
  $: progressBar = (() => {
    const filled = Math.floor(timerProgress * 10);
    return '█'.repeat(filled) + '░'.repeat(10 - filled);
  })();
</script>

<footer class="status-bar">
  <div class="left">
    <span class="connection" class:connected={$connected}>
      {$connected ? '🟢 Conectado' : '🔴 Desconectado'}
    </span>
    {#if showNewEmailBadge}
      <span class="new-email-badge" class:has-new={$newEmailCount > 0}>
        {#if $newEmailCount > 0}
          📬 {$newEmailCount} {$newEmailCount === 1 ? 'NOVO!' : 'NOVOS!'}
        {:else}
          ✓ 0 novos
        {/if}
      </span>
    {/if}
  </div>

  <div class="center">
    {#if $syncing}
      <span class="syncing">
        <span class="spinner"></span>
        Sincronizando...
      </span>
    {:else}
      <button class="sync-btn" on:click={syncEmails} title="Sincronizar (r)">
        🔄 Sync
      </button>
      {#if $autoRefreshEnabled}
        <span class="timer" title="Auto-refresh em {remainingSeconds}s">
          ⏱ <span class="progress-bar">{progressBar}</span> {remainingSeconds}s
        </span>
      {:else}
        <span class="last-sync">Último: {formatLastSync($lastSync)}</span>
      {/if}
    {/if}
  </div>

  <div class="right">
    <ThemeToggle />
    <div class="divider"></div>
    <button class="icon-btn" on:click={switchToTerminal} title="Abrir Terminal (T)">
      <span class="icon">⌨</span>
    </button>
    <button class="icon-btn" on:click={toggleDebug} title="Debug (D)">
      <span class="icon">🐛</span>
    </button>
    <span class="shortcuts">
      <kbd>/</kbd> busca
      <kbd>?</kbd> ajuda
      <kbd>c</kbd> compor
    </span>
  </div>
</footer>

<style>
  .status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-xs) var(--space-md);
    background: var(--bg-secondary);
    border-top: 1px solid var(--border-color);
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .left, .center, .right {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .connection {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
  }

  .connection.connected {
    color: var(--accent-success);
  }

  .syncing {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    color: var(--accent-primary);
  }

  .spinner {
    display: inline-block;
    width: 12px;
    height: 12px;
    border: 2px solid var(--border-color);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .sync-btn {
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-xs);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .sync-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
  }

  .last-sync {
    color: var(--text-muted);
  }

  .timer {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    color: var(--text-muted);
    font-family: monospace;
    font-size: 11px;
  }

  .progress-bar {
    color: var(--accent-primary);
    letter-spacing: -1px;
  }

  .new-email-badge {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: bold;
    font-size: 11px;
  }

  .new-email-badge.has-new {
    background: var(--accent-success);
    color: var(--bg-primary);
    animation: pulse 0.5s ease-in-out infinite alternate;
  }

  @keyframes pulse {
    from { opacity: 1; transform: scale(1); }
    to { opacity: 0.8; transform: scale(1.05); }
  }

  .shortcuts {
    display: flex;
    gap: var(--space-md);
  }

  kbd {
    display: inline-block;
    padding: 1px 4px;
    font-family: monospace;
    background: var(--bg-tertiary);
    border-radius: 3px;
    margin-right: 2px;
  }

  .icon-btn {
    padding: 2px 6px;
    font-size: 14px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .icon-btn:hover {
    background: var(--bg-hover);
    border-color: var(--border-color);
  }

  .icon-btn .icon {
    display: block;
  }

  .divider {
    width: 1px;
    height: 16px;
    background: var(--border-color);
  }
</style>
