<script>
  import { createEventDispatcher } from 'svelte';
  import { fly } from 'svelte/transition';
  import { quintOut } from 'svelte/easing';
  import {
    selectAll,
    clearSelection,
    invertSelection,
    selectUnread,
    selectRead,
    selectWithAttachments,
    selectStarred,
    selectToday,
    selectThisWeek,
    selectOlderThanWeek,
    selectBySender,
    someSelected,
    selectedCount
  } from '../stores/selection.js';
  import { selectedEmail } from '../stores/emails.js';

  export var show = false;
  export var anchorX = 0;
  export var anchorY = 0;

  var dispatch = createEventDispatcher();

  function close() {
    show = false;
    dispatch('close');
  }

  function handleAction(action) {
    action();
    close();
  }

  function handleClickOutside(e) {
    if (show) {
      close();
    }
  }

  // Get current sender for "select same sender" option
  $: currentSender = $selectedEmail?.fromEmail || null;
</script>

<svelte:window on:click={handleClickOutside} />

{#if show}
  <div
    class="smart-menu"
    style="left: {anchorX}px; top: {anchorY}px;"
    transition:fly={{ y: -10, duration: 200, easing: quintOut }}
    on:click|stopPropagation
  >
    <div class="menu-header">
      <span class="menu-title">Seleção Inteligente</span>
      {#if $someSelected}
        <span class="selected-badge">{$selectedCount}</span>
      {/if}
    </div>

    <div class="menu-section">
      <div class="section-title">Básico</div>
      <button class="menu-item" on:click={() => handleAction(selectAll)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
          <path d="M9 12l2 2 4-4"/>
        </svg>
        <span>Selecionar todos</span>
        <kbd>Ctrl+A</kbd>
      </button>
      <button class="menu-item" on:click={() => handleAction(invertSelection)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M4 4v16h16"/>
          <path d="M20 20V4H4"/>
        </svg>
        <span>Inverter seleção</span>
      </button>
      <button class="menu-item" on:click={() => handleAction(clearSelection)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
        </svg>
        <span>Limpar seleção</span>
        <kbd>Esc</kbd>
      </button>
    </div>

    <div class="menu-divider"></div>

    <div class="menu-section">
      <div class="section-title">Por Status</div>
      <button class="menu-item" on:click={() => handleAction(selectUnread)}>
        <svg viewBox="0 0 24 24" fill="currentColor">
          <circle cx="12" cy="12" r="4"/>
        </svg>
        <span>Não lidos</span>
      </button>
      <button class="menu-item" on:click={() => handleAction(selectRead)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="4"/>
        </svg>
        <span>Lidos</span>
      </button>
      <button class="menu-item" on:click={() => handleAction(selectStarred)}>
        <svg viewBox="0 0 24 24" fill="#facc15" stroke="#facc15" stroke-width="2">
          <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
        </svg>
        <span>Favoritos</span>
      </button>
      <button class="menu-item" on:click={() => handleAction(selectWithAttachments)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48"/>
        </svg>
        <span>Com anexos</span>
      </button>
    </div>

    <div class="menu-divider"></div>

    <div class="menu-section">
      <div class="section-title">Por Data</div>
      <button class="menu-item" on:click={() => handleAction(selectToday)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="4" width="18" height="18" rx="2"/>
          <line x1="16" y1="2" x2="16" y2="6"/>
          <line x1="8" y1="2" x2="8" y2="6"/>
          <line x1="3" y1="10" x2="21" y2="10"/>
        </svg>
        <span>Hoje</span>
      </button>
      <button class="menu-item" on:click={() => handleAction(selectThisWeek)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="4" width="18" height="18" rx="2"/>
          <line x1="16" y1="2" x2="16" y2="6"/>
          <line x1="8" y1="2" x2="8" y2="6"/>
          <line x1="3" y1="10" x2="21" y2="10"/>
          <line x1="8" y1="14" x2="8" y2="14"/>
          <line x1="12" y1="14" x2="12" y2="14"/>
          <line x1="16" y1="14" x2="16" y2="14"/>
        </svg>
        <span>Esta semana</span>
      </button>
      <button class="menu-item" on:click={() => handleAction(selectOlderThanWeek)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <polyline points="12 6 12 12 16 14"/>
        </svg>
        <span>Mais de 1 semana</span>
      </button>
    </div>

    {#if currentSender}
      <div class="menu-divider"></div>

      <div class="menu-section">
        <div class="section-title">Por Remetente</div>
        <button class="menu-item sender" on:click={() => handleAction(() => selectBySender(currentSender))}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
            <circle cx="12" cy="7" r="4"/>
          </svg>
          <span class="sender-text">
            <span class="sender-label">Mesmo remetente</span>
            <span class="sender-email">{currentSender}</span>
          </span>
        </button>
      </div>
    {/if}
  </div>
{/if}

<style>
  .smart-menu {
    position: fixed;
    z-index: 2000;
    min-width: 240px;
    background: rgba(30, 35, 45, 0.98);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    box-shadow:
      0 8px 32px rgba(0, 0, 0, 0.4),
      0 0 0 1px rgba(255, 255, 255, 0.05) inset;
    overflow: hidden;
  }

  .menu-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  }

  .menu-title {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: rgba(255, 255, 255, 0.5);
  }

  .selected-badge {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 20px;
    height: 20px;
    padding: 0 6px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border-radius: 10px;
    font-size: 11px;
    font-weight: 700;
    color: white;
  }

  .menu-section {
    padding: 6px;
  }

  .section-title {
    padding: 6px 10px 4px;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: rgba(255, 255, 255, 0.35);
  }

  .menu-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.08);
    margin: 0;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 10px;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: rgba(255, 255, 255, 0.8);
    font-size: 13px;
    text-align: left;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .menu-item:hover {
    background: rgba(255, 255, 255, 0.08);
    color: white;
  }

  .menu-item:active {
    background: rgba(255, 255, 255, 0.12);
  }

  .menu-item svg {
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    opacity: 0.7;
  }

  .menu-item:hover svg {
    opacity: 1;
  }

  .menu-item span {
    flex: 1;
  }

  .menu-item kbd {
    padding: 2px 6px;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 4px;
    font-family: inherit;
    font-size: 10px;
    color: rgba(255, 255, 255, 0.4);
  }

  .menu-item.sender {
    padding: 10px;
  }

  .sender-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .sender-label {
    font-size: 13px;
  }

  .sender-email {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.5);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 160px;
  }
</style>
