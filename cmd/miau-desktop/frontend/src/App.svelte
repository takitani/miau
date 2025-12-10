<script>
  import { onMount } from 'svelte';
  import EmailList from './lib/components/EmailList.svelte';
  import FolderList from './lib/components/FolderList.svelte';
  import EmailViewer from './lib/components/EmailViewer.svelte';
  import SearchPanel from './lib/components/SearchPanel.svelte';
  import SearchResultModal from './lib/components/SearchResultModal.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import DebugPanel from './lib/components/DebugPanel.svelte';
  import HelpOverlay from './lib/components/HelpOverlay.svelte';
  import AIChat from './lib/components/AIChat.svelte';
  import ComposeModal from './lib/components/ComposeModal.svelte';
  import AnalyticsPanel from './lib/components/AnalyticsPanel.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import SelectionBar from './lib/components/SelectionBar.svelte';
  import ModernSidebar from './lib/components/ModernSidebar.svelte';
  import LayoutToggle from './lib/components/LayoutToggle.svelte';
  import CalendarPanel from './lib/components/CalendarPanel.svelte';
  import CalendarEventModal from './lib/components/CalendarEventModal.svelte';
  import AuthOverlay from './lib/components/AuthOverlay.svelte';
  import { emails, selectedEmail, loadEmails, currentFolder } from './lib/stores/emails.js';
  import { folders, loadFolders } from './lib/stores/folders.js';
  import { showSearch, showHelp, showAI, showCompose, showAnalytics, showSettings, aiWithContext, activePanel, setupKeyboardShortcuts, connect, syncEssentialFolders, showThreadView, threadEmailId, closeThreadView } from './lib/stores/ui.js';
  import { showCalendarPanel } from './lib/stores/calendar.js';
  import ThreadView from './lib/components/ThreadView.svelte';
  import { debugEnabled, info, setupDebugEvents } from './lib/stores/debug.js';
  import { layoutMode, initLayoutPreferences } from './lib/stores/layout.js';
  import { NeedsOAuth2Auth, StartOAuth2Auth } from './lib/wailsjs/wailsjs/go/desktop/App.js';

  // Auth state
  var needsAuth = false;
  var authInProgress = false;
  var authError = null;

  async function checkAuth() {
    try {
      needsAuth = await NeedsOAuth2Auth();
      if (needsAuth) {
        info('OAuth2 authentication required...');
        authInProgress = true;
        authError = null;
        try {
          await StartOAuth2Auth();
          needsAuth = false;
          authInProgress = false;
          info('Authentication successful!');
          // Reload app after auth
          window.location.reload();
        } catch (err) {
          authError = err.message || 'Authentication failed';
          authInProgress = false;
        }
      }
    } catch (err) {
      console.error('Failed to check auth:', err);
    }
  }

  // Get email context for AI
  $: emailContext = $aiWithContext && $selectedEmail ? $selectedEmail : null;

  // Panel sizes (pixels for folders, percentage for emails)
  var STORAGE_KEY = 'miau-panel-sizes';
  var DEFAULT_FOLDERS_WIDTH = 200;
  var DEFAULT_EMAILS_WIDTH = 400;
  var MIN_FOLDERS_WIDTH = 120;
  var MAX_FOLDERS_WIDTH = 350;
  var MIN_EMAILS_WIDTH = 250;
  var MAX_EMAILS_WIDTH = 800;

  var foldersWidth = DEFAULT_FOLDERS_WIDTH;
  var emailsWidth = DEFAULT_EMAILS_WIDTH;
  var draggingDivider = null;
  var startX = 0;
  var startWidth = 0;

  // Load saved sizes
  function loadPanelSizes() {
    try {
      var saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        var sizes = JSON.parse(saved);
        foldersWidth = sizes.foldersWidth || DEFAULT_FOLDERS_WIDTH;
        emailsWidth = sizes.emailsWidth || DEFAULT_EMAILS_WIDTH;
      }
    } catch (e) {
      console.error('Failed to load panel sizes:', e);
    }
  }

  // Save sizes to localStorage
  function savePanelSizes() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        foldersWidth,
        emailsWidth
      }));
    } catch (e) {
      console.error('Failed to save panel sizes:', e);
    }
  }

  // Start dragging a divider
  function startDrag(divider, e) {
    draggingDivider = divider;
    startX = e.clientX;
    startWidth = divider === 'folders' ? foldersWidth : emailsWidth;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  }

  // Handle mouse move during drag
  function handleMouseMove(e) {
    if (!draggingDivider) return;

    var delta = e.clientX - startX;

    if (draggingDivider === 'folders') {
      var newWidth = startWidth + delta;
      foldersWidth = Math.max(MIN_FOLDERS_WIDTH, Math.min(MAX_FOLDERS_WIDTH, newWidth));
    } else if (draggingDivider === 'emails') {
      var newWidth = startWidth + delta;
      emailsWidth = Math.max(MIN_EMAILS_WIDTH, Math.min(MAX_EMAILS_WIDTH, newWidth));
    }
  }

  // Stop dragging
  function handleMouseUp() {
    if (draggingDivider) {
      draggingDivider = null;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      savePanelSizes();
    }
  }

  // Double-click to reset
  function resetDivider(divider) {
    if (divider === 'folders') {
      foldersWidth = DEFAULT_FOLDERS_WIDTH;
    } else {
      emailsWidth = DEFAULT_EMAILS_WIDTH;
    }
    savePanelSizes();
  }

  // Initialize app
  onMount(async () => {
    loadPanelSizes();
    initLayoutPreferences();

    info('App initializing...');
    setupKeyboardShortcuts();
    setupDebugEvents();

    // Check if OAuth2 auth is needed first
    await checkAuth();
    if (needsAuth) return; // Wait for auth to complete

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

<svelte:window on:mousemove={handleMouseMove} on:mouseup={handleMouseUp} />

<main class="app">
  <!-- Auth Overlay (blocks everything when authenticating) -->
  {#if needsAuth || authInProgress}
    <AuthOverlay inProgress={authInProgress} error={authError} on:retry={checkAuth} />
  {/if}

  <!-- Overlays -->
  {#if $showSearch}
    <SearchPanel />
  {/if}

  <!-- Search Result Modal (for emails not in current list) -->
  <SearchResultModal />

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

  <!-- Calendar Event Modal -->
  <CalendarEventModal />

  {#if $layoutMode === 'modern'}
    <!-- Modern Layout: Sidebar + 2 Panels -->
    <div class="layout modern">
      <!-- Modern Sidebar with Tasks, AI, Folders -->
      <ModernSidebar folders={$folders} selectedFolder={$currentFolder} on:select />

      <!-- Divider: Sidebar | Emails -->
      <div
        class="divider"
        class:dragging={draggingDivider === 'emails'}
        on:mousedown={(e) => startDrag('emails', e)}
        on:dblclick={() => resetDivider('emails')}
        role="separator"
        aria-orientation="vertical"
        tabindex="0"
        title="Arrastar para redimensionar (duplo-clique para resetar)"
      ></div>

      <!-- Email List Panel -->
      <section class="emails-panel" class:active={$activePanel === 'emails'} style="width: {emailsWidth}px">
        <EmailList />
      </section>

      <!-- Divider 2: Emails | Viewer -->
      <div
        class="divider"
        class:dragging={draggingDivider === 'viewer'}
        on:mousedown={(e) => startDrag('viewer', e)}
        on:dblclick={() => resetDivider('viewer')}
        role="separator"
        aria-orientation="vertical"
        tabindex="0"
        title="Arrastar para redimensionar (duplo-clique para resetar)"
      ></div>

      <!-- Email Viewer Panel / Analytics Panel / Thread View / Calendar Panel -->
      <section class="viewer-panel" class:active={$activePanel === 'viewer'}>
        {#if $showCalendarPanel}
          <CalendarPanel />
        {:else if $showThreadView && $threadEmailId}
          <ThreadView emailId={$threadEmailId} on:close={closeThreadView} />
        {:else if $showAnalytics}
          <AnalyticsPanel />
        {:else if $selectedEmail}
          <EmailViewer email={$selectedEmail} />
        {:else}
          <div class="empty-state">
            <p>Selecione um email para visualizar</p>
            <p class="hint">Use j/k para navegar, Enter ou t para thread, p para analytics</p>
          </div>
        {/if}
      </section>
    </div>
  {:else}
    <!-- Legacy Layout: 3 Panels (Folders | Emails | Viewer) -->
    <div class="layout legacy">
      <!-- Folders Panel -->
      <aside class="folders-panel" class:active={$activePanel === 'folders'} style="width: {foldersWidth}px">
        <FolderList />
      </aside>

      <!-- Divider 1: Folders | Emails -->
      <div
        class="divider"
        class:dragging={draggingDivider === 'folders'}
        on:mousedown={(e) => startDrag('folders', e)}
        on:dblclick={() => resetDivider('folders')}
        role="separator"
        aria-orientation="vertical"
        tabindex="0"
        title="Arrastar para redimensionar (duplo-clique para resetar)"
      ></div>

      <!-- Email List Panel -->
      <section class="emails-panel" class:active={$activePanel === 'emails'} style="width: {emailsWidth}px">
        <EmailList />
      </section>

      <!-- Divider 2: Emails | Viewer -->
      <div
        class="divider"
        class:dragging={draggingDivider === 'emails'}
        on:mousedown={(e) => startDrag('emails', e)}
        on:dblclick={() => resetDivider('emails')}
        role="separator"
        aria-orientation="vertical"
        tabindex="0"
        title="Arrastar para redimensionar (duplo-clique para resetar)"
      ></div>

      <!-- Email Viewer Panel / Analytics Panel / Thread View / Calendar Panel -->
      <section class="viewer-panel" class:active={$activePanel === 'viewer'}>
        {#if $showCalendarPanel}
          <CalendarPanel />
        {:else if $showThreadView && $threadEmailId}
          <ThreadView emailId={$threadEmailId} on:close={closeThreadView} />
        {:else if $showAnalytics}
          <AnalyticsPanel />
        {:else if $selectedEmail}
          <EmailViewer email={$selectedEmail} />
        {:else}
          <div class="empty-state">
            <p>Selecione um email para visualizar</p>
            <p class="hint">Use j/k para navegar, Enter ou t para thread, p para analytics</p>
          </div>
        {/if}
      </section>
    </div>
  {/if}

  <!-- Status Bar -->
  <StatusBar />

  <!-- Layout Toggle (floating bottom-right) -->
  <LayoutToggle />

  <!-- Selection Bar (floating at bottom when emails selected) -->
  <SelectionBar />

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
    flex-shrink: 0;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .folders-panel.active {
    background: var(--bg-active);
  }

  .emails-panel {
    flex-shrink: 0;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .emails-panel.active {
    background: var(--bg-active);
  }

  .viewer-panel {
    flex: 1;
    min-width: 200px;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .viewer-panel.active {
    background: var(--bg-active);
  }

  /* Resizable divider */
  .divider {
    width: 4px;
    background: var(--border-color);
    cursor: col-resize;
    flex-shrink: 0;
    transition: background 0.15s ease;
    position: relative;
  }

  .divider:hover,
  .divider.dragging {
    background: var(--accent-primary);
  }

  .divider::before {
    content: '';
    position: absolute;
    top: 0;
    left: -4px;
    right: -4px;
    bottom: 0;
    /* Larger hit area for easier grabbing */
  }

  .divider:focus {
    outline: none;
    background: var(--accent-primary);
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

  /* Modern layout transitions */
  .layout.modern {
    animation: fadeIn 200ms ease;
  }

  .layout.legacy {
    animation: fadeIn 200ms ease;
  }

  @keyframes fadeIn {
    from {
      opacity: 0.8;
    }
    to {
      opacity: 1;
    }
  }
</style>
