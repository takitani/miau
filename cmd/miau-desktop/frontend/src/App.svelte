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
  import { emails, selectedEmail, loadEmails, currentFolder } from './lib/stores/emails.js';
  import { folders, loadFolders } from './lib/stores/folders.js';
  import { showSearch, showHelp, showAI, showCompose, showAnalytics, aiWithContext, activePanel, setupKeyboardShortcuts, connect } from './lib/stores/ui.js';
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

    info('Loading emails from ' + $currentFolder);
    await loadEmails($currentFolder);

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
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .folders-panel {
    width: 200px;
    border-right: 1px solid var(--border-color);
    overflow-y: auto;
    flex-shrink: 0;
  }

  .folders-panel.active {
    background: var(--bg-active);
  }

  .emails-panel {
    flex: 1;
    min-width: 300px;
    border-right: 1px solid var(--border-color);
    overflow-y: auto;
  }

  .emails-panel.active {
    background: var(--bg-active);
  }

  .viewer-panel {
    flex: 1.5;
    overflow-y: auto;
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
