<script>
  import { onMount } from 'svelte';
  import EmailList from './lib/components/EmailList.svelte';
  import FolderList from './lib/components/FolderList.svelte';
  import EmailViewer from './lib/components/EmailViewer.svelte';
  import SearchPanel from './lib/components/SearchPanel.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import DebugPanel from './lib/components/DebugPanel.svelte';
  import HelpOverlay from './lib/components/HelpOverlay.svelte';
  import AIChat from './lib/components/AIChat.svelte';
  import ComposeModal from './lib/components/ComposeModal.svelte';
  import AnalyticsPanel from './lib/components/AnalyticsPanel.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import { emails, selectedEmail, loadEmails, currentFolder } from './lib/stores/emails.js';
  import { folders, loadFolders } from './lib/stores/folders.js';
  import { showSearch, showHelp, showAI, showCompose, showAnalytics, showSettings, aiWithContext, activePanel, setupKeyboardShortcuts, connect, syncEssentialFolders } from './lib/stores/ui.js';
  import { debugEnabled, info, setupDebugEvents } from './lib/stores/debug.js';

  // Get email context for AI
  $: emailContext = $aiWithContext && $selectedEmail ? $selectedEmail : null;

  // Initialize app
  onMount(async () => {
    info('App initializing...');
    setupKeyboardShortcuts();
    setupDebugEvents();

    info('Connecting to IMAP server...');
    await connect();

    info('Loading folders...');
    await loadFolders();

    info('Loading emails from cache...');
    await loadEmails($currentFolder);

    info('Starting initial sync (INBOX, Sent, Trash)...');
    await syncEssentialFolders();

    info('App ready. Press ? for help, D for debug.');
  });
</script>

<main class="app">
  <!-- Overlays -->
  {#if $showSearch}
    <SearchPanel />
  {/if}

  {#if $showHelp}
    <HelpOverlay />
  {/if}

  {#if $showAI}
    <AIChat emailContext={emailContext} />
  {/if}

  {#if $showCompose}
    <ComposeModal />
  {/if}

  {#if $showSettings}
    <SettingsModal />
  {/if}

  <div class="layout">
    <!-- Folders Panel -->
    <aside class="folders-panel" class:active={$activePanel === 'folders'}>
      <FolderList />
    </aside>

    <!-- Email List Panel -->
    <section class="emails-panel" class:active={$activePanel === 'emails'}>
      <EmailList />
    </section>

    <!-- Email Viewer Panel / Analytics Panel -->
    <section class="viewer-panel" class:active={$activePanel === 'viewer'}>
      {#if $showAnalytics}
        <AnalyticsPanel />
      {:else if $selectedEmail}
        <EmailViewer email={$selectedEmail} />
      {:else}
        <div class="empty-state">
          <p>Selecione um email para visualizar</p>
          <p class="hint">Use j/k para navegar, Enter para abrir, p para analytics</p>
        </div>
      {/if}
    </section>
  </div>

  <!-- Status Bar -->
  <StatusBar />

  <!-- Debug Panel -->
  {#if $debugEnabled}
    <DebugPanel />
  {/if}
</main>

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  .layout {
    display: grid;
    grid-template-columns: 200px 400px 1fr;
    flex: 1;
    overflow: hidden;
  }

  .folders-panel {
    grid-column: 1;
    border-right: 1px solid var(--border-color);
    overflow-y: auto;
    overflow-x: hidden;
  }

  .folders-panel.active {
    background: var(--bg-active);
  }

  .emails-panel {
    grid-column: 2;
    border-right: 1px solid var(--border-color);
    overflow-y: auto;
    overflow-x: hidden;
  }

  .emails-panel.active {
    background: var(--bg-active);
  }

  .viewer-panel {
    grid-column: 3;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .viewer-panel.active {
    background: var(--bg-active);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-muted);
  }

  .empty-state .hint {
    font-size: 0.875rem;
    margin-top: 0.5rem;
    opacity: 0.7;
  }
</style>
