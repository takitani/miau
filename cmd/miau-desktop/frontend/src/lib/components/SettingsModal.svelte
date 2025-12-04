<script>
  import { onMount } from 'svelte';
  import { showSettings } from '../stores/ui.js';
  import { info, error as logError } from '../stores/debug.js';

  var activeTab = 'folders';
  var loading = true;
  var saving = false;

  // Settings state
  var availableFolders = [];
  var settings = {
    syncFolders: [],
    uiTheme: 'dark',
    uiShowPreview: true,
    uiPageSize: 50,
    composeFormat: 'html',
    composeSendDelay: 30,
    syncInterval: '5m'
  };

  var tabs = [
    { id: 'folders', label: 'Folders' },
    { id: 'ui', label: 'UI' },
    { id: 'compose', label: 'Compose' },
    { id: 'sync', label: 'Sync' },
    { id: 'about', label: 'About' }
  ];

  onMount(async () => {
    await loadSettings();
  });

  async function loadSettings() {
    loading = true;
    try {
      if (window.go?.desktop?.App) {
        // Load current settings
        var loaded = await window.go.desktop.App.GetSettings();
        if (loaded) {
          settings = { ...settings, ...loaded };
        }

        // Load available folders with selection status
        var folders = await window.go.desktop.App.GetAvailableFolders();
        if (folders) {
          availableFolders = folders;
        }
      }
    } catch (err) {
      logError('Failed to load settings', err);
    } finally {
      loading = false;
    }
  }

  async function saveSettings() {
    saving = true;
    try {
      if (window.go?.desktop?.App) {
        // Update syncFolders from checkbox state
        settings.syncFolders = availableFolders
          .filter(f => f.isSelected)
          .map(f => f.name);

        await window.go.desktop.App.SaveSettings(settings);
        info('Settings saved successfully');
        close();
      }
    } catch (err) {
      logError('Failed to save settings', err);
    } finally {
      saving = false;
    }
  }

  function close() {
    showSettings.set(false);
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    }
  }

  function toggleFolder(folderName) {
    availableFolders = availableFolders.map(f =>
      f.name === folderName ? { ...f, isSelected: !f.isSelected } : f
    );
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="overlay" on:click={close} role="button" tabindex="-1" on:keydown={handleKeydown}>
  <div class="settings-modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="settings-header">
      <h2>Settings</h2>
      <button class="close-btn" on:click={close}>X</button>
    </div>

    <div class="tabs">
      {#each tabs as tab}
        <button
          class="tab"
          class:active={activeTab === tab.id}
          on:click={() => activeTab = tab.id}
        >
          {tab.label}
        </button>
      {/each}
    </div>

    <div class="settings-content">
      {#if loading}
        <div class="loading">Loading settings...</div>
      {:else if activeTab === 'folders'}
        <div class="tab-content">
          <h3>Folders to Sync</h3>
          <p class="hint">Select which folders to sync on startup and auto-refresh.</p>
          <div class="folder-list">
            {#each availableFolders as folder}
              <label class="folder-item">
                <input
                  type="checkbox"
                  checked={folder.isSelected}
                  on:change={() => toggleFolder(folder.name)}
                />
                <span>{folder.name}</span>
              </label>
            {/each}
          </div>
        </div>
      {:else if activeTab === 'ui'}
        <div class="tab-content">
          <h3>User Interface</h3>

          <div class="setting-row">
            <label for="theme">Theme</label>
            <select id="theme" bind:value={settings.uiTheme}>
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </div>

          <div class="setting-row">
            <label for="preview">Show Preview</label>
            <input
              type="checkbox"
              id="preview"
              bind:checked={settings.uiShowPreview}
            />
          </div>

          <div class="setting-row">
            <label for="pageSize">Page Size</label>
            <input
              type="number"
              id="pageSize"
              bind:value={settings.uiPageSize}
              min="10"
              max="200"
            />
          </div>
        </div>
      {:else if activeTab === 'compose'}
        <div class="tab-content">
          <h3>Email Composition</h3>

          <div class="setting-row">
            <label for="format">Default Format</label>
            <select id="format" bind:value={settings.composeFormat}>
              <option value="html">HTML</option>
              <option value="plain">Plain Text</option>
            </select>
          </div>

          <div class="setting-row">
            <label for="sendDelay">Send Delay (seconds)</label>
            <input
              type="number"
              id="sendDelay"
              bind:value={settings.composeSendDelay}
              min="0"
              max="60"
            />
            <span class="hint">Time to cancel before email is sent</span>
          </div>
        </div>
      {:else if activeTab === 'sync'}
        <div class="tab-content">
          <h3>Synchronization</h3>

          <div class="setting-row">
            <label for="interval">Auto-refresh Interval</label>
            <select id="interval" bind:value={settings.syncInterval}>
              <option value="1m">1 minute</option>
              <option value="2m">2 minutes</option>
              <option value="5m">5 minutes</option>
              <option value="10m">10 minutes</option>
              <option value="15m">15 minutes</option>
              <option value="30m">30 minutes</option>
            </select>
          </div>
        </div>
      {:else if activeTab === 'about'}
        <div class="tab-content about">
          <h3>miau</h3>
          <p class="version">Mail Intelligence Assistant Utility</p>
          <p class="description">
            A local email client with TUI interface via IMAP
            and AI integration for email management assistance.
          </p>
          <div class="links">
            <p>Privacy-focused - everything runs locally</p>
          </div>
        </div>
      {/if}
    </div>

    <div class="settings-footer">
      <button class="btn btn-secondary" on:click={close}>Cancel</button>
      <button class="btn btn-primary" on:click={saveSettings} disabled={saving}>
        {saving ? 'Saving...' : 'Save'}
      </button>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .settings-modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    width: 90%;
    max-width: 600px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .settings-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .settings-header h2 {
    margin: 0;
    font-size: var(--font-lg);
    font-weight: 600;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: var(--radius-sm);
  }

  .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .tabs {
    display: flex;
    border-bottom: 1px solid var(--border-color);
    padding: 0 var(--space-md);
  }

  .tab {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    padding: var(--space-sm) var(--space-md);
    cursor: pointer;
    font-size: var(--font-sm);
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }

  .tab:hover {
    color: var(--text-primary);
  }

  .tab.active {
    color: var(--accent-primary);
    border-bottom-color: var(--accent-primary);
  }

  .settings-content {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-lg);
  }

  .loading {
    text-align: center;
    color: var(--text-muted);
    padding: var(--space-xl);
  }

  .tab-content h3 {
    margin: 0 0 var(--space-sm) 0;
    font-size: var(--font-md);
    font-weight: 600;
  }

  .hint {
    font-size: var(--font-xs);
    color: var(--text-muted);
    margin: var(--space-xs) 0 var(--space-md) 0;
  }

  .folder-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
    max-height: 300px;
    overflow-y: auto;
  }

  .folder-item {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
    cursor: pointer;
  }

  .folder-item:hover {
    background: var(--bg-hover);
  }

  .folder-item input[type="checkbox"] {
    cursor: pointer;
  }

  .folder-item span {
    font-size: var(--font-sm);
  }

  .setting-row {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-bottom: var(--space-md);
  }

  .setting-row label {
    min-width: 150px;
    font-size: var(--font-sm);
  }

  .setting-row select,
  .setting-row input[type="number"] {
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: var(--font-sm);
  }

  .setting-row input[type="number"] {
    width: 80px;
  }

  .setting-row input[type="checkbox"] {
    cursor: pointer;
    width: 18px;
    height: 18px;
  }

  .about {
    text-align: center;
  }

  .about .version {
    color: var(--text-muted);
    font-size: var(--font-sm);
    margin-top: var(--space-xs);
  }

  .about .description {
    margin-top: var(--space-md);
    font-size: var(--font-sm);
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .about .links {
    margin-top: var(--space-lg);
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .settings-footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-sm);
    padding: var(--space-md) var(--space-lg);
    border-top: 1px solid var(--border-color);
  }

  .btn {
    padding: var(--space-xs) var(--space-md);
    border-radius: var(--radius-sm);
    font-size: var(--font-sm);
    cursor: pointer;
    border: 1px solid transparent;
  }

  .btn-secondary {
    background: var(--bg-tertiary);
    border-color: var(--border-color);
    color: var(--text-primary);
  }

  .btn-secondary:hover {
    background: var(--bg-hover);
  }

  .btn-primary {
    background: var(--accent-primary);
    color: white;
  }

  .btn-primary:hover {
    opacity: 0.9;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
