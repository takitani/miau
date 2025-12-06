<script>
  import { sidebarExpanded, toggleSidebar, SIDEBAR_COLLAPSED_WIDTH, SIDEBAR_EXPANDED_WIDTH } from '../stores/layout.js';
  import TasksWidget from './TasksWidget.svelte';
  import FolderList from './FolderList.svelte';

  export let folders = [];
  export let selectedFolder = 'INBOX';

  var sectionsExpanded = {
    tasks: true,
    ai: true,
    folders: true
  };

  function toggleSection(section) {
    sectionsExpanded[section] = !sectionsExpanded[section];
  }
</script>

<aside
  class="modern-sidebar"
  class:expanded={$sidebarExpanded}
  class:collapsed={!$sidebarExpanded}
  style="width: {$sidebarExpanded ? SIDEBAR_EXPANDED_WIDTH : SIDEBAR_COLLAPSED_WIDTH}px"
>
  <!-- Header with toggle -->
  <div class="sidebar-header">
    <button class="toggle-btn" on:click={toggleSidebar} title={$sidebarExpanded ? 'Collapse sidebar ([)' : 'Expand sidebar ([)'}>
      {#if $sidebarExpanded}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M11 17l-5-5 5-5M18 17l-5-5 5-5"/>
        </svg>
      {:else}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M13 17l5-5-5-5M6 17l5-5-5-5"/>
        </svg>
      {/if}
    </button>
    {#if $sidebarExpanded}
      <span class="sidebar-title">miau</span>
    {/if}
  </div>

  <div class="sidebar-content">
    <!-- Tasks Section -->
    <section class="sidebar-section">
      {#if $sidebarExpanded}
        <button class="section-header" on:click={() => toggleSection('tasks')}>
          <span class="section-icon">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 11l3 3L22 4"/>
              <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
            </svg>
          </span>
          <span class="section-title">Tasks</span>
          <span class="section-chevron" class:rotated={!sectionsExpanded.tasks}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </span>
        </button>
        {#if sectionsExpanded.tasks}
          <div class="section-content">
            <TasksWidget />
          </div>
        {/if}
      {:else}
        <button class="icon-btn" title="Tasks">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 11l3 3L22 4"/>
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
          </svg>
        </button>
      {/if}
    </section>

    <!-- AI Suggestions Section -->
    <section class="sidebar-section">
      {#if $sidebarExpanded}
        <button class="section-header" on:click={() => toggleSection('ai')}>
          <span class="section-icon">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 16v-4M12 8h.01"/>
            </svg>
          </span>
          <span class="section-title">AI Suggestions</span>
          <span class="section-chevron" class:rotated={!sectionsExpanded.ai}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </span>
        </button>
        {#if sectionsExpanded.ai}
          <div class="section-content ai-section">
            <div class="ai-placeholder">
              <span class="ai-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
                </svg>
              </span>
              <span class="ai-text">No suggestions yet</span>
              <span class="ai-hint">AI will suggest tasks based on your emails</span>
            </div>
          </div>
        {/if}
      {:else}
        <button class="icon-btn" title="AI Suggestions">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 16v-4M12 8h.01"/>
          </svg>
        </button>
      {/if}
    </section>

    <!-- Folders Section -->
    <section class="sidebar-section folders-section">
      {#if $sidebarExpanded}
        <button class="section-header" on:click={() => toggleSection('folders')}>
          <span class="section-icon">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
          </span>
          <span class="section-title">Folders</span>
          <span class="section-chevron" class:rotated={!sectionsExpanded.folders}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </span>
        </button>
        {#if sectionsExpanded.folders}
          <div class="section-content">
            <FolderList {folders} {selectedFolder} compact={true} on:select />
          </div>
        {/if}
      {:else}
        <button class="icon-btn" title="Folders">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </button>
      {/if}
    </section>
  </div>
</aside>

<style>
  .modern-sidebar {
    display: flex;
    flex-direction: column;
    background: var(--bg-secondary);
    border-right: 1px solid var(--border-color);
    transition: width var(--sidebar-transition);
    overflow: hidden;
    flex-shrink: 0;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm);
    border-bottom: 1px solid var(--border-color);
    min-height: 48px;
  }

  .toggle-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    border-radius: var(--radius-md);
    cursor: pointer;
    transition: all var(--transition-fast);
    flex-shrink: 0;
  }

  .toggle-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .sidebar-title {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--accent-primary);
    letter-spacing: -0.5px;
  }

  .sidebar-content {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    padding: var(--space-xs);
  }

  .sidebar-section {
    margin-bottom: var(--space-xs);
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-sm);
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--font-sm);
    font-weight: 500;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
    text-align: left;
  }

  .section-header:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .section-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  .section-title {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .section-chevron {
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform var(--transition-fast);
  }

  .section-chevron.rotated {
    transform: rotate(-90deg);
  }

  .section-content {
    padding: 0 var(--space-xs);
    animation: slideDown 150ms ease;
  }

  @keyframes slideDown {
    from {
      opacity: 0;
      transform: translateY(-8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 44px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .icon-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
  }

  .folders-section {
    flex: 1;
    display: flex;
    flex-direction: column;
  }

  .folders-section .section-content {
    flex: 1;
    overflow-y: auto;
  }

  /* AI placeholder */
  .ai-section {
    padding: var(--space-sm);
  }

  .ai-placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-md);
    text-align: center;
    color: var(--text-muted);
  }

  .ai-icon {
    color: var(--text-muted);
    opacity: 0.5;
  }

  .ai-text {
    font-size: var(--font-sm);
    color: var(--text-secondary);
  }

  .ai-hint {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  /* Collapsed state */
  .collapsed .sidebar-content {
    padding: var(--space-xs) 0;
  }

  .collapsed .icon-btn {
    margin: var(--space-xs) var(--space-xs);
    width: calc(100% - var(--space-md));
  }
</style>
