<script>
  import { onMount } from 'svelte';
  import { showSearch } from '../stores/ui.js';
  import { searchEmails, clearSearch, searchQuery, isSearching } from '../stores/emails.js';

  let query = '';
  let inputEl;
  let searchTimeout;

  // Focus input on mount
  onMount(() => {
    inputEl?.focus();
    // If already searching, show current query
    query = $searchQuery || '';
  });

  // Debounced search as user types
  function handleInput() {
    clearTimeout(searchTimeout);
    if (query.length >= 2) {
      searchTimeout = setTimeout(() => searchEmails(query), 200);
    } else if (query.length === 0 && $isSearching) {
      clearSearch();
    }
  }

  // Handle keyboard
  function handleKeydown(e) {
    if (e.key === 'Escape') {
      if (query) {
        // First Esc clears query
        query = '';
        clearSearch();
      } else {
        // Second Esc closes search bar
        close();
      }
      e.preventDefault();
    } else if (e.key === 'Enter') {
      // Immediate search on Enter
      if (query.length >= 2) {
        clearTimeout(searchTimeout);
        searchEmails(query);
      }
      e.preventDefault();
    }
  }

  function clear() {
    query = '';
    clearSearch();
    inputEl?.focus();
  }

  function close() {
    showSearch.set(false);
    if ($isSearching) {
      clearSearch();
    }
  }
</script>

<div class="search-bar">
  <span class="search-icon">🔍</span>
  <input
    bind:this={inputEl}
    bind:value={query}
    type="text"
    placeholder="Buscar emails... (min 2 caracteres)"
    class="search-input"
    on:input={handleInput}
    on:keydown={handleKeydown}
  />
  {#if query}
    <button class="clear-btn" on:click={clear} title="Limpar busca">✕</button>
  {/if}
  <button class="close-btn" on:click={close} title="Fechar (Esc)">
    <span class="close-icon">✕</span>
  </button>
  {#if $isSearching}
    <span class="search-indicator">Buscando: "{$searchQuery}"</span>
  {/if}
</div>

<style>
  .search-bar {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border-color);
  }

  .search-icon {
    font-size: var(--font-md);
    opacity: 0.6;
  }

  .search-input {
    flex: 1;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    padding: var(--space-xs) var(--space-sm);
    font-size: var(--font-sm);
    color: var(--text-primary);
    min-width: 200px;
  }

  .search-input:focus {
    outline: none;
    border-color: var(--accent-color);
  }

  .search-input::placeholder {
    color: var(--text-muted);
  }

  .clear-btn, .close-btn {
    padding: var(--space-xs);
    color: var(--text-muted);
    background: transparent;
    border: none;
    cursor: pointer;
    border-radius: var(--radius-sm);
    font-size: var(--font-sm);
  }

  .clear-btn:hover, .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .search-indicator {
    font-size: var(--font-xs);
    color: var(--accent-color);
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }
</style>
