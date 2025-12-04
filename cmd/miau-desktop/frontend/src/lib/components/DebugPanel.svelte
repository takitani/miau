<script>
  import { logs, clearLogs, debugEnabled, LogLevel } from '../stores/debug.js';

  function getLevelClass(level) {
    switch (level) {
      case LogLevel.ERROR: return 'error';
      case LogLevel.WARN: return 'warn';
      case LogLevel.EVENT: return 'event';
      case LogLevel.INFO: return 'info';
      default: return 'debug';
    }
  }

  function formatData(data) {
    if (!data) return '';
    try {
      return JSON.stringify(data, null, 2);
    } catch {
      return String(data);
    }
  }

  function close() {
    debugEnabled.set(false);
  }
</script>

<div class="debug-panel">
  <div class="debug-header">
    <h3>🐛 Debug Console</h3>
    <div class="debug-actions">
      <button onclick={clearLogs}>Clear</button>
      <button onclick={close}>✕</button>
    </div>
  </div>

  <div class="debug-logs">
    {#each $logs as entry (entry.id)}
      <div class="log-entry {getLevelClass(entry.level)}">
        <span class="timestamp">{entry.timestamp}</span>
        <span class="level">[{entry.level.toUpperCase()}]</span>
        <span class="message">{entry.message}</span>
        {#if entry.data}
          <pre class="data">{formatData(entry.data)}</pre>
        {/if}
      </div>
    {:else}
      <div class="empty">No logs yet. Press D to toggle this panel.</div>
    {/each}
  </div>

  <div class="debug-footer">
    <span>{$logs.length} entries</span>
    <span class="hint">Press D to close</span>
  </div>
</div>

<style>
  .debug-panel {
    position: fixed;
    bottom: 24px;
    right: 8px;
    width: 500px;
    max-height: 400px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    display: flex;
    flex-direction: column;
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 11px;
    z-index: 1000;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }

  .debug-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-tertiary);
    border-radius: 8px 8px 0 0;
  }

  .debug-header h3 {
    margin: 0;
    font-size: 12px;
    font-weight: 600;
  }

  .debug-actions {
    display: flex;
    gap: 8px;
  }

  .debug-actions button {
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 11px;
  }

  .debug-actions button:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .debug-logs {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
  }

  .log-entry {
    padding: 4px 8px;
    border-radius: 4px;
    margin-bottom: 4px;
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: flex-start;
  }

  .log-entry.error {
    background: rgba(239, 68, 68, 0.15);
    color: #f87171;
  }

  .log-entry.warn {
    background: rgba(234, 179, 8, 0.15);
    color: #fbbf24;
  }

  .log-entry.event {
    background: rgba(168, 85, 247, 0.15);
    color: #c084fc;
  }

  .log-entry.info {
    background: rgba(59, 130, 246, 0.15);
    color: #60a5fa;
  }

  .log-entry.debug {
    background: rgba(107, 114, 128, 0.1);
    color: var(--text-muted);
  }

  .timestamp {
    color: var(--text-muted);
    font-size: 10px;
  }

  .level {
    font-weight: 600;
    min-width: 50px;
  }

  .message {
    flex: 1;
  }

  .data {
    width: 100%;
    margin: 4px 0 0 0;
    padding: 4px 8px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: 4px;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .empty {
    color: var(--text-muted);
    text-align: center;
    padding: 20px;
  }

  .debug-footer {
    display: flex;
    justify-content: space-between;
    padding: 4px 12px;
    border-top: 1px solid var(--border-color);
    color: var(--text-muted);
    font-size: 10px;
  }

  .hint {
    opacity: 0.7;
  }
</style>
