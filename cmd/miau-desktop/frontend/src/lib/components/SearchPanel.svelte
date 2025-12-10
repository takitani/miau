<script>
  import { onMount } from 'svelte';
  import { showSearch, openSearchResultModal } from '../stores/ui.js';
  import { selectEmail } from '../stores/emails.js';

  let query = '';
  let results = [];
  let selectedIndex = 0;
  let loading = false;
  let inputEl;

  // Focus input on mount
  onMount(() => {
    inputEl?.focus();
  });

  // Debounced search
  let searchTimeout;
  $: if (query.length >= 2) {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => search(query), 150);
  } else {
    results = [];
  }

  async function search(q) {
    loading = true;
    try {
      if (window.go?.desktop?.App) {
        const result = await window.go.desktop.App.Search(q, 20);
        results = result?.emails || [];
      } else {
        // Mock for development
        results = getMockResults(q);
      }
      selectedIndex = 0;
    } catch (err) {
      console.error('Search failed:', err);
      results = [];
    } finally {
      loading = false;
    }
  }

  // Handle keyboard navigation
  function handleKeydown(e) {
    switch (e.key) {
      case 'Escape':
        close();
        break;
      case 'ArrowDown':
        e.preventDefault();
        selectedIndex = Math.min(selectedIndex + 1, results.length - 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        selectedIndex = Math.max(selectedIndex - 1, 0);
        break;
      case 'Enter':
        e.preventDefault();
        if (results[selectedIndex]) {
          selectResult(results[selectedIndex]);
        }
        break;
    }
  }

  // Select a result
  function selectResult(email) {
    // Try to select in current list first
    const found = selectEmail(email.id);
    if (!found) {
      // Email not in list - open in modal
      openSearchResultModal(email.id);
    }
    close();
  }

  // Close search panel
  function close() {
    showSearch.set(false);
    query = '';
    results = [];
  }

  // Format date
  function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' });
  }

  // Mock results for development
  function getMockResults(q) {
    return [
      { id: 1, subject: `Result 1 for "${q}"`, fromName: 'Test User', date: new Date().toISOString() },
      { id: 2, subject: `Another result for "${q}"`, fromName: 'Someone', date: new Date().toISOString() }
    ];
  }
</script>

<div class="search-overlay" on:click|self={close}>
  <div class="search-panel">
    <div class="search-input-wrapper">
      <span class="search-icon">🔍</span>
      <input
        bind:this={inputEl}
        bind:value={query}
        type="text"
        placeholder="Buscar emails..."
        class="search-input"
        on:keydown={handleKeydown}
      />
      {#if query}
        <button class="clear-btn" on:click={() => query = ''}>✕</button>
      {/if}
    </div>

    {#if loading}
      <div class="loading">Buscando...</div>
    {:else if results.length > 0}
      <ul class="results">
        {#each results as result, index (result.id)}
          <li>
            <button
              class="result-item"
              class:selected={index === selectedIndex}
              on:click={() => selectResult(result)}
              on:mouseenter={() => selectedIndex = index}
            >
              <span class="from">{result.fromName || result.fromEmail}</span>
              <span class="subject truncate">{result.subject}</span>
              <span class="date">{formatDate(result.date)}</span>
            </button>
          </li>
        {/each}
      </ul>
    {:else if query.length >= 2}
      <div class="no-results">Nenhum resultado para "{query}"</div>
    {:else}
      <div class="hint">Digite pelo menos 2 caracteres para buscar</div>
    {/if}

    <div class="shortcuts">
      <span><kbd>↑↓</kbd> navegar</span>
      <span><kbd>Enter</kbd> selecionar</span>
      <span><kbd>Esc</kbd> fechar</span>
    </div>
  </div>
</div>

<style>
  .search-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 100px;
    z-index: 1000;
  }

  .search-panel {
    width: 100%;
    max-width: 600px;
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-lg);
    overflow: hidden;
  }

  .search-input-wrapper {
    display: flex;
    align-items: center;
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-color);
  }

  .search-icon {
    margin-right: var(--space-sm);
    font-size: var(--font-lg);
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    font-size: var(--font-lg);
    color: var(--text-primary);
  }

  .search-input::placeholder {
    color: var(--text-muted);
  }

  .clear-btn {
    padding: var(--space-xs);
    color: var(--text-muted);
    font-size: var(--font-sm);
  }

  .clear-btn:hover {
    color: var(--text-primary);
  }

  .loading, .no-results, .hint {
    padding: var(--space-lg);
    text-align: center;
    color: var(--text-muted);
  }

  .results {
    list-style: none;
    max-height: 400px;
    overflow-y: auto;
  }

  .result-item {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-sm) var(--space-md);
    text-align: left;
    color: var(--text-secondary);
    transition: background var(--transition-fast);
  }

  .result-item:hover,
  .result-item.selected {
    background: var(--bg-hover);
  }

  .result-item.selected {
    background: var(--bg-selected);
  }

  .result-item .from {
    width: 150px;
    flex-shrink: 0;
    font-size: var(--font-sm);
    color: var(--text-primary);
  }

  .result-item .subject {
    flex: 1;
    font-size: var(--font-sm);
  }

  .result-item .date {
    flex-shrink: 0;
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .shortcuts {
    display: flex;
    gap: var(--space-md);
    justify-content: center;
    padding: var(--space-sm);
    border-top: 1px solid var(--border-color);
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .shortcuts kbd {
    display: inline-block;
    padding: 2px 6px;
    font-family: monospace;
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
    margin-right: var(--space-xs);
  }
</style>
