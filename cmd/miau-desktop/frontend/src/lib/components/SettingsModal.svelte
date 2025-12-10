<script>
  import { onMount, onDestroy } from 'svelte';
  import { showSettings } from '../stores/ui.js';
  import { info, error as logError } from '../stores/debug.js';
  import { syncContacts, syncStatus, contactsSyncing, loadSyncStatus } from '../stores/contacts.js';

  var activeTab = 'folders';
  var loading = true;
  var saving = false;
  var syncingThreads = false;
  var threadSyncResult = null;
  var threadSyncProgress = null; // { phase, processed, total, found, page }
  var unsubscribeProgress = null;
  var contactSyncResult = null;

  // Basecamp state
  var basecampConfig = {
    enabled: false,
    clientId: '',
    clientSecret: '',
    accountId: '',
    connected: false
  };
  var basecampAuthenticating = false;
  var basecampConnecting = false;
  var basecampAccounts = []; // Available accounts after auth
  var basecampResult = null;

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
    { id: 'basecamp', label: 'Basecamp' },
    { id: 'about', label: 'About' }
  ];

  onMount(async () => {
    await loadSettings();
    await loadSyncStatus();

    // Listen for thread sync progress events
    if (window.runtime?.EventsOn) {
      unsubscribeProgress = window.runtime.EventsOn('thread-sync-progress', (data) => {
        threadSyncProgress = data;
      });
    }
  });

  async function handleContactSync(fullSync) {
    contactSyncResult = null;
    try {
      await syncContacts(fullSync);
      contactSyncResult = { success: true };
    } catch (err) {
      contactSyncResult = { success: false, error: err.message || String(err) };
    }
  }

  // Basecamp functions
  async function loadBasecampConfig() {
    try {
      if (window.go?.desktop?.App?.GetBasecampConfig) {
        var cfg = await window.go.desktop.App.GetBasecampConfig();
        if (cfg) {
          basecampConfig = {
            enabled: cfg.enabled || false,
            clientId: cfg.clientId || '',
            clientSecret: cfg.clientSecret || '',
            accountId: cfg.accountId || '',
            connected: cfg.connected || false
          };
        }
      }
    } catch (err) {
      logError('Failed to load Basecamp config', err);
    }
  }

  async function saveBasecampConfig() {
    try {
      if (window.go?.desktop?.App?.SaveBasecampConfig) {
        await window.go.desktop.App.SaveBasecampConfig(basecampConfig);
        basecampResult = { success: true, message: 'Configuration saved' };
      }
    } catch (err) {
      basecampResult = { success: false, error: err.message || String(err) };
    }
  }

  async function authenticateBasecamp() {
    basecampAuthenticating = true;
    basecampResult = null;
    basecampAccounts = [];
    try {
      if (window.go?.desktop?.App?.AuthenticateBasecamp) {
        var accounts = await window.go.desktop.App.AuthenticateBasecamp();
        basecampAccounts = accounts || [];
        if (basecampAccounts.length === 0) {
          basecampResult = { success: false, error: 'No Basecamp accounts found' };
        } else if (basecampAccounts.length === 1) {
          // Auto-select if only one account
          await selectBasecampAccount(basecampAccounts[0].id);
        } else {
          basecampResult = { success: true, message: 'Select a Basecamp account' };
        }
      }
    } catch (err) {
      basecampResult = { success: false, error: err.message || String(err) };
    } finally {
      basecampAuthenticating = false;
    }
  }

  async function selectBasecampAccount(accountId) {
    basecampConnecting = true;
    basecampResult = null;
    try {
      if (window.go?.desktop?.App?.SelectBasecampAccount) {
        await window.go.desktop.App.SelectBasecampAccount(accountId);
        basecampConfig.connected = true;
        basecampConfig.accountId = String(accountId);
        basecampAccounts = [];
        basecampResult = { success: true, message: 'Connected to Basecamp!' };
        await loadBasecampConfig();
      }
    } catch (err) {
      basecampResult = { success: false, error: err.message || String(err) };
    } finally {
      basecampConnecting = false;
    }
  }

  async function connectBasecamp() {
    basecampConnecting = true;
    basecampResult = null;
    try {
      if (window.go?.desktop?.App?.ConnectBasecamp) {
        await window.go.desktop.App.ConnectBasecamp();
        basecampConfig.connected = true;
        basecampResult = { success: true, message: 'Connected to Basecamp!' };
      }
    } catch (err) {
      basecampResult = { success: false, error: err.message || String(err) };
    } finally {
      basecampConnecting = false;
    }
  }

  async function disconnectBasecamp() {
    try {
      if (window.go?.desktop?.App?.DisconnectBasecamp) {
        await window.go.desktop.App.DisconnectBasecamp();
        basecampConfig.connected = false;
        basecampResult = { success: true, message: 'Disconnected from Basecamp' };
      }
    } catch (err) {
      basecampResult = { success: false, error: err.message || String(err) };
    }
  }

  onDestroy(() => {
    if (unsubscribeProgress) {
      unsubscribeProgress();
    }
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

        // Load Basecamp config
        await loadBasecampConfig();
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
    // Cancel any ongoing thread sync when closing
    if (syncingThreads) {
      cancelThreadSync();
    }
    showSettings.set(false);
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    }
  }

  async function cancelThreadSync() {
    if (window.go?.desktop?.App?.CancelThreadSync) {
      await window.go.desktop.App.CancelThreadSync();
      syncingThreads = false;
      threadSyncProgress = null;
      threadSyncResult = { success: false, error: 'Cancelled by user' };
    }
  }

  function toggleFolder(folderName) {
    availableFolders = availableFolders.map(f =>
      f.name === folderName ? { ...f, isSelected: !f.isSelected } : f
    );
  }

  async function syncThreadsFromGmail() {
    syncingThreads = true;
    threadSyncResult = null;
    threadSyncProgress = null;
    try {
      var updated = await window.go.desktop.App.SyncThreadsFromGmail();
      threadSyncResult = { success: true, count: updated };
      info(`Thread sync completed: ${updated} emails updated`);
    } catch (err) {
      threadSyncResult = { success: false, error: err.message || String(err) };
      logError('Thread sync failed', err);
    } finally {
      syncingThreads = false;
      threadSyncProgress = null;
    }
  }

  // Calculate estimated time remaining
  function getEstimatedTime(processed, total) {
    if (!processed || !total || processed === 0) return '';
    var remaining = total - processed;
    var secondsRemaining = Math.ceil(remaining / 100 * 0.15); // ~150ms per batch of 100
    if (secondsRemaining < 60) return `~${secondsRemaining}s remaining`;
    var minutes = Math.ceil(secondsRemaining / 60);
    return `~${minutes}min remaining`;
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

          <div class="sync-section">
            <h4>Thread Sync from Gmail</h4>
            <p class="hint">
              Synchronize thread IDs from Gmail API to ensure accurate message grouping.
              This uses Gmail's native thread detection which is more reliable than local algorithms.
            </p>
            <div class="sync-action">
              <button
                class="btn btn-secondary"
                on:click={syncThreadsFromGmail}
                disabled={syncingThreads}
              >
                {syncingThreads ? 'Syncing...' : 'Sync Threads from Gmail'}
              </button>
            </div>

            {#if syncingThreads && threadSyncProgress}
              <div class="progress-section">
                {#if threadSyncProgress.phase === 'listing'}
                  <div class="progress-label">
                    <span>Listing messages... Page {threadSyncProgress.page}</span>
                    <span class="progress-eta">~{threadSyncProgress.found?.toLocaleString()} total</span>
                  </div>
                  <div class="progress-bar indeterminate">
                    <div class="progress-fill"></div>
                  </div>
                {:else if threadSyncProgress.phase === 'fetching'}
                  <div class="progress-label">
                    <span>Fetching thread IDs: {threadSyncProgress.processed?.toLocaleString()} / {threadSyncProgress.total?.toLocaleString()}</span>
                    <span class="progress-eta">{getEstimatedTime(threadSyncProgress.processed, threadSyncProgress.total)}</span>
                  </div>
                  <div class="progress-bar">
                    <div
                      class="progress-fill"
                      style="width: {(threadSyncProgress.processed / threadSyncProgress.total * 100).toFixed(1)}%"
                    ></div>
                  </div>
                  <div class="progress-percent">
                    {(threadSyncProgress.processed / threadSyncProgress.total * 100).toFixed(1)}%
                  </div>
                {/if}
                <button class="btn btn-cancel" on:click={cancelThreadSync}>
                  Cancel
                </button>
              </div>
            {/if}

            {#if threadSyncResult}
              <div class="sync-result" class:success={threadSyncResult.success} class:error={!threadSyncResult.success}>
                {#if threadSyncResult.success}
                  ✓ Updated {threadSyncResult.count?.toLocaleString()} email(s)
                {:else}
                  ✗ Error: {threadSyncResult.error}
                {/if}
              </div>
            {/if}
          </div>

          <div class="sync-section">
            <h4>Contacts Sync from Gmail</h4>
            <p class="hint">
              Synchronize contacts from Google Contacts to enable autocomplete when composing emails.
            </p>

            {#if $syncStatus}
              <div class="sync-status-info">
                <span class="status-label">Status:</span>
                <span class="status-value">{$syncStatus.status || 'never_synced'}</span>
                {#if $syncStatus.totalContacts > 0}
                  <span class="status-count">({$syncStatus.totalContacts} contacts)</span>
                {/if}
              </div>
            {/if}

            <div class="sync-action">
              <button
                class="btn btn-secondary"
                on:click={() => handleContactSync(false)}
                disabled={$contactsSyncing}
              >
                {$contactsSyncing ? 'Syncing...' : 'Sync Contacts'}
              </button>
              <button
                class="btn btn-outline"
                on:click={() => handleContactSync(true)}
                disabled={$contactsSyncing}
                title="Full sync (slower, but more thorough)"
              >
                Full Sync
              </button>
            </div>

            {#if contactSyncResult}
              <div class="sync-result" class:success={contactSyncResult.success} class:error={!contactSyncResult.success}>
                {#if contactSyncResult.success}
                  ✓ Contacts synchronized successfully
                {:else}
                  ✗ Error: {contactSyncResult.error}
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {:else if activeTab === 'basecamp'}
        <div class="tab-content">
          <h3>Basecamp Integration</h3>
          <p class="hint">
            Connect to Basecamp to manage projects and to-dos.
            Register your app at <a href="https://launchpad.37signals.com/integrations" target="_blank">launchpad.37signals.com</a>
          </p>

          <div class="setting-row">
            <label for="bc-enabled">Enable Basecamp</label>
            <input
              type="checkbox"
              id="bc-enabled"
              bind:checked={basecampConfig.enabled}
            />
          </div>

          {#if basecampConfig.enabled}
            <div class="setting-row">
              <label for="bc-client-id">Client ID</label>
              <input
                type="text"
                id="bc-client-id"
                bind:value={basecampConfig.clientId}
                placeholder="Your Basecamp Client ID"
                class="text-input"
              />
            </div>

            <div class="setting-row">
              <label for="bc-client-secret">Client Secret</label>
              <input
                type="password"
                id="bc-client-secret"
                bind:value={basecampConfig.clientSecret}
                placeholder="Your Basecamp Client Secret"
                class="text-input"
              />
            </div>

            <div class="sync-action">
              <button
                class="btn btn-secondary"
                on:click={saveBasecampConfig}
              >
                Save Credentials
              </button>
            </div>

            {#if basecampConfig.clientId && basecampConfig.clientSecret}
              <div class="sync-section">
                <h4>Connection</h4>

                {#if basecampConfig.connected}
                  <div class="sync-status-info">
                    <span class="status-label">Status:</span>
                    <span class="status-value connected">Connected</span>
                    {#if basecampConfig.accountId}
                      <span class="status-count">(Account: {basecampConfig.accountId})</span>
                    {/if}
                  </div>
                  <div class="sync-action">
                    <button
                      class="btn btn-secondary"
                      on:click={connectBasecamp}
                      disabled={basecampConnecting}
                    >
                      {basecampConnecting ? 'Reconnecting...' : 'Reconnect'}
                    </button>
                    <button
                      class="btn btn-outline"
                      on:click={disconnectBasecamp}
                    >
                      Disconnect
                    </button>
                  </div>
                {:else}
                  <div class="sync-status-info">
                    <span class="status-label">Status:</span>
                    <span class="status-value">Not connected</span>
                  </div>
                  <div class="sync-action">
                    <button
                      class="btn btn-primary"
                      on:click={authenticateBasecamp}
                      disabled={basecampAuthenticating}
                    >
                      {basecampAuthenticating ? 'Authenticating...' : 'Connect to Basecamp'}
                    </button>
                  </div>
                {/if}

                {#if basecampAccounts.length > 0}
                  <div class="account-selector">
                    <p class="hint">Select a Basecamp account:</p>
                    {#each basecampAccounts as account}
                      <button
                        class="btn btn-account"
                        on:click={() => selectBasecampAccount(account.id)}
                        disabled={basecampConnecting}
                      >
                        {account.name}
                      </button>
                    {/each}
                  </div>
                {/if}

                {#if basecampResult}
                  <div class="sync-result" class:success={basecampResult.success} class:error={!basecampResult.success}>
                    {#if basecampResult.success}
                      ✓ {basecampResult.message}
                    {:else}
                      ✗ Error: {basecampResult.error}
                    {/if}
                  </div>
                {/if}
              </div>
            {/if}
          {/if}
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

  .sync-section {
    margin-top: var(--space-lg);
    padding-top: var(--space-lg);
    border-top: 1px solid var(--border-color);
  }

  .sync-section h4 {
    margin: 0 0 var(--space-xs) 0;
    font-size: var(--font-sm);
    font-weight: 600;
  }

  .sync-action {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    margin-top: var(--space-sm);
  }

  .sync-result {
    font-size: var(--font-sm);
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
  }

  .sync-result.success {
    background: rgba(34, 197, 94, 0.2);
    color: #22c55e;
  }

  .sync-result.error {
    background: rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .progress-section {
    margin-top: var(--space-md);
    padding: var(--space-sm);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }

  .progress-label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: var(--font-xs);
    color: var(--text-secondary);
    margin-bottom: var(--space-xs);
  }

  .progress-eta {
    color: var(--text-muted);
  }

  .progress-bar {
    height: 8px;
    background: var(--bg-primary);
    border-radius: 4px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: var(--accent-primary);
    border-radius: 4px;
    transition: width 0.3s ease;
  }

  .progress-bar.indeterminate .progress-fill {
    width: 30%;
    animation: indeterminate 1.5s ease-in-out infinite;
  }

  @keyframes indeterminate {
    0% { transform: translateX(-100%); }
    100% { transform: translateX(400%); }
  }

  .progress-percent {
    font-size: var(--font-xs);
    color: var(--accent-primary);
    text-align: right;
    margin-top: var(--space-xs);
    font-weight: 600;
  }

  .btn-cancel {
    margin-top: var(--space-sm);
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-xs);
  }

  .btn-cancel:hover {
    background: rgba(239, 68, 68, 0.2);
    border-color: #ef4444;
    color: #ef4444;
  }

  .sync-result {
    margin-top: var(--space-md);
    font-size: var(--font-sm);
    padding: var(--space-sm);
    border-radius: var(--radius-sm);
  }

  .sync-status-info {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    font-size: var(--font-sm);
    margin: var(--space-sm) 0;
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }

  .status-label {
    color: var(--text-muted);
  }

  .status-value {
    color: var(--text-primary);
    font-weight: 500;
  }

  .status-count {
    color: var(--text-muted);
    font-size: var(--font-xs);
  }

  .btn-outline {
    background: transparent;
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
  }

  .btn-outline:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .btn-outline:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Basecamp styles */
  .text-input {
    flex: 1;
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: var(--font-sm);
  }

  .text-input::placeholder {
    color: var(--text-muted);
  }

  .status-value.connected {
    color: #22c55e;
    font-weight: 600;
  }

  .account-selector {
    margin-top: var(--space-md);
    padding: var(--space-sm);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }

  .btn-account {
    display: block;
    width: 100%;
    margin-top: var(--space-xs);
    padding: var(--space-sm);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    color: var(--text-primary);
    text-align: left;
    cursor: pointer;
    border-radius: var(--radius-sm);
  }

  .btn-account:hover {
    background: var(--bg-hover);
    border-color: var(--accent-primary);
  }

  .hint a {
    color: var(--accent-primary);
    text-decoration: none;
  }

  .hint a:hover {
    text-decoration: underline;
  }
</style>
