<script>
  import { fly, scale } from 'svelte/transition';
  import { elasticOut, quintOut } from 'svelte/easing';
  import {
    selectedCount,
    someSelected,
    allSelected,
    selectAll,
    clearSelection,
    invertSelection,
    exitSelectionMode,
    batchArchive,
    batchDelete,
    batchMarkRead,
    batchMarkUnread,
    batchStar
  } from '../stores/selection.js';

  var showSmartMenu = false;

  function handleKeydown(e) {
    if (!$someSelected) return;

    // Batch operations shortcuts
    switch (e.key) {
      case 'Escape':
        exitSelectionMode();
        e.preventDefault();
        break;
      case 'e':
        if (!e.ctrlKey && !e.metaKey) {
          batchArchive();
          e.preventDefault();
        }
        break;
      case 'x':
      case '#':
        if (!e.ctrlKey && !e.metaKey) {
          batchDelete();
          e.preventDefault();
        }
        break;
      case 'r':
        if (!e.ctrlKey && !e.metaKey && !e.shiftKey) {
          batchMarkRead();
          e.preventDefault();
        }
        break;
      case 'u':
        if (!e.ctrlKey && !e.metaKey) {
          batchMarkUnread();
          e.preventDefault();
        }
        break;
      case 's':
        if (!e.ctrlKey && !e.metaKey) {
          batchStar();
          e.preventDefault();
        }
        break;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if $someSelected}
  <div
    class="selection-bar"
    transition:fly={{ y: 100, duration: 400, easing: quintOut }}
  >
    <div class="bar-content">
      <!-- Selection count with pulse animation -->
      <div class="selection-info">
        <span class="count-badge" in:scale={{ duration: 300, easing: elasticOut }}>
          {$selectedCount}
        </span>
        <span class="count-label">
          {$selectedCount === 1 ? 'email selecionado' : 'emails selecionados'}
        </span>
      </div>

      <!-- Divider -->
      <div class="divider"></div>

      <!-- Quick actions -->
      <div class="actions">
        <button
          class="action-btn"
          on:click={batchArchive}
          title="Arquivar (e)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 8v13H3V8M1 3h22v5H1zM10 12h4"/>
          </svg>
          <span class="action-label">Arquivar</span>
        </button>

        <button
          class="action-btn danger"
          on:click={batchDelete}
          title="Excluir (x)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
          </svg>
          <span class="action-label">Excluir</span>
        </button>

        <button
          class="action-btn"
          on:click={batchMarkRead}
          title="Marcar como lido (r)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span class="action-label">Lido</span>
        </button>

        <button
          class="action-btn"
          on:click={batchMarkUnread}
          title="Marcar como não lido (u)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19m-6.72-1.07a3 3 0 11-4.24-4.24"/>
            <line x1="1" y1="1" x2="23" y2="23"/>
          </svg>
          <span class="action-label">Não lido</span>
        </button>

        <button
          class="action-btn star"
          on:click={batchStar}
          title="Favoritar (s)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
          </svg>
          <span class="action-label">Favoritar</span>
        </button>
      </div>

      <!-- Divider -->
      <div class="divider"></div>

      <!-- Selection actions -->
      <div class="selection-actions">
        <button
          class="select-btn"
          on:click={selectAll}
          class:active={$allSelected}
          title="Selecionar todos (Ctrl+A)"
        >
          {$allSelected ? 'Todos' : 'Selecionar todos'}
        </button>

        <button
          class="select-btn"
          on:click={invertSelection}
          title="Inverter seleção"
        >
          Inverter
        </button>
      </div>

      <!-- Close button -->
      <button
        class="close-btn"
        on:click={exitSelectionMode}
        title="Cancelar (Esc)"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

  </div>
{/if}

<style>
  .selection-bar {
    position: fixed;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 1000;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .bar-content {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 20px;
    background: rgba(30, 35, 45, 0.95);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 16px;
    box-shadow:
      0 8px 32px rgba(0, 0, 0, 0.4),
      0 0 0 1px rgba(255, 255, 255, 0.05) inset;
  }

  .selection-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .count-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 28px;
    height: 28px;
    padding: 0 8px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border-radius: 14px;
    font-size: 14px;
    font-weight: 700;
    color: white;
    box-shadow: 0 2px 8px rgba(102, 126, 234, 0.4);
  }

  .count-label {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.7);
    white-space: nowrap;
  }

  .divider {
    width: 1px;
    height: 24px;
    background: rgba(255, 255, 255, 0.15);
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .action-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.8);
    font-size: 12px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .action-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .action-btn:active {
    transform: scale(0.95);
  }

  .action-btn.danger:hover {
    background: rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .action-btn.star:hover {
    background: rgba(250, 204, 21, 0.2);
    color: #facc15;
  }

  .action-btn svg {
    width: 16px;
    height: 16px;
  }

  .action-label {
    font-weight: 500;
  }

  .selection-actions {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .select-btn {
    padding: 6px 10px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: rgba(255, 255, 255, 0.7);
    font-size: 11px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .select-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.2);
    color: white;
  }

  .select-btn.active {
    background: rgba(102, 126, 234, 0.3);
    border-color: rgba(102, 126, 234, 0.5);
    color: #a5b4fc;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: rgba(255, 255, 255, 0.05);
    border: none;
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.5);
    cursor: pointer;
    transition: all 0.2s ease;
    margin-left: 4px;
  }

  .close-btn:hover {
    background: rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .close-btn svg {
    width: 16px;
    height: 16px;
  }

  /* Responsive */
  @media (max-width: 800px) {
    .action-label {
      display: none;
    }
  }
</style>
