<script>
  import { createEventDispatcher } from 'svelte';
  import { folders as foldersStore, selectFolder, foldersLoading } from '../stores/folders.js';
  import { currentFolder } from '../stores/emails.js';

  // Props for flexibility (can override store or use directly)
  export let folders = null; // If provided, use instead of store
  export let selectedFolder = null; // If provided, use instead of store
  export let compact = false; // Compact mode for sidebar

  var dispatch = createEventDispatcher();

  // Use props or store
  $: folderList = folders || $foldersStore;
  $: selected = selectedFolder || $currentFolder;

  // Folder icon mapping (use simple emojis without variation selectors)
  const folderIcons = {
    'INBOX': '📥',
    '[Gmail]/Sent Mail': '📤',
    '[Gmail]/E-mails enviados': '📤',
    '[Gmail]/Drafts': '📝',
    '[Gmail]/Rascunhos': '📝',
    '[Gmail]/Starred': '⭐',
    '[Gmail]/Com estrela': '⭐',
    '[Gmail]/Trash': '🗑',
    '[Gmail]/Lixeira': '🗑',
    '[Gmail]/Spam': '⚠',
    '[Gmail]/All Mail': '📁',
    '[Gmail]/Todos os e-mails': '📁',
    '[Gmail]/Important': '❗',
    '[Gmail]/Importante': '❗',
  };

  // Get folder icon
  function getIcon(name) {
    return folderIcons[name] || '📁';
  }

  // Format folder name
  function formatName(name) {
    return name.replace('[Gmail]/', '').replace('[Google]/', '');
  }

  // Handle folder click
  function handleClick(name) {
    selectFolder(name);
    dispatch('select', { folder: name });
  }
</script>

<nav class="folder-list" class:compact>
  {#if !compact}
    <header class="list-header">
      <h3>Pastas</h3>
    </header>
  {/if}

  {#if $foldersLoading}
    <div class="loading">Carregando...</div>
  {:else}
    <ul class="folders">
      {#each folderList as folder (folder.id)}
        <li>
          <button
            class="folder-item"
            class:selected={selected === folder.name}
            on:click={() => handleClick(folder.name)}
            title={folder.name}
          >
            <span class="icon">{getIcon(folder.name)}</span>
            {#if !compact}
              <span class="name truncate">{formatName(folder.name)}</span>
            {/if}
            {#if folder.unreadMessages > 0}
              <span class="badge" class:compact>{folder.unreadMessages}</span>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</nav>

<style>
  .folder-list {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .list-header {
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-secondary);
  }

  .list-header h3 {
    font-size: var(--font-sm);
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .loading {
    padding: var(--space-md);
    color: var(--text-muted);
    text-align: center;
  }

  .folders {
    list-style: none;
    padding: var(--space-sm) 0;
  }

  .folder-item {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-sm) var(--space-md);
    text-align: left;
    color: var(--text-secondary);
    transition: all var(--transition-fast);
  }

  .folder-item:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .folder-item.selected {
    background: var(--bg-selected);
    color: var(--accent-primary);
  }

  .icon {
    font-size: var(--font-md);
    flex-shrink: 0;
  }

  .name {
    flex: 1;
    font-size: var(--font-sm);
  }

  .badge {
    background: var(--accent-primary);
    color: var(--bg-primary);
    font-size: var(--font-xs);
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 10px;
    min-width: 20px;
    text-align: center;
  }

  /* Compact mode styles */
  .folder-list.compact {
    height: auto;
  }

  .compact .folders {
    padding: 0;
  }

  .compact .folder-item {
    padding: var(--space-xs) var(--space-sm);
    justify-content: flex-start;
    gap: var(--space-sm);
  }

  .compact .icon {
    font-size: var(--font-sm);
  }

  .compact .badge {
    font-size: 10px;
    padding: 1px 4px;
    min-width: 16px;
  }
</style>
