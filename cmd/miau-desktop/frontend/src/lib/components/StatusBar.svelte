<script>
  import { connected, syncing, lastSync, syncEmails, switchToTerminal } from '../stores/ui.js';
  import { toggleDebug } from '../stores/debug.js';

  // Format last sync time
  function formatLastSync(date) {
    if (!date) return 'Nunca';
    const diff = Date.now() - date.getTime();

    if (diff < 60000) return 'Agora';
    if (diff < 3600000) return `${Math.floor(diff / 60000)} min atrás`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h atrás`;
    return date.toLocaleDateString('pt-BR');
  }
</script>

<footer class="status-bar">
  <div class="left">
    <span class="connection" class:connected={$connected}>
      {$connected ? '🟢 Conectado' : '🔴 Desconectado'}
    </span>
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
      <span class="last-sync">Último: {formatLastSync($lastSync)}</span>
    {/if}
  </div>

  <div class="right">
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
</style>
