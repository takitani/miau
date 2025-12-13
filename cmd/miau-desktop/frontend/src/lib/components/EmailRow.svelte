<script>
  import { createEventDispatcher } from 'svelte';
  import { scale } from 'svelte/transition';
  import {
    selectedIds,
    selectionMode,
    toggleSelection,
    selectRange,
    lastSelectedIndex,
    dragSelecting,
    startDragSelection,
    updateDragSelection,
    endDragSelection
  } from '../stores/selection.js';

  export var email;
  export var selected = false; // Cursor selection (current row)
  export var index = 0;

  var dispatch = createEventDispatcher();
  var hovering = false;

  // Is this email checked (multi-select)?
  $: isChecked = $selectedIds.has(email.id);

  // Show checkbox when: hovering, selection mode active, or this email is checked
  $: showCheckbox = hovering || $selectionMode || isChecked;

  // Format date
  function formatDate(dateStr) {
    var date = new Date(dateStr);
    var now = new Date();
    var diff = now - date;

    // Today: show time
    if (diff < 86400000 && date.getDate() === now.getDate()) {
      return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
    }

    // This year: show day/month
    if (date.getFullYear() === now.getFullYear()) {
      return date.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' });
    }

    // Other: show full date
    return date.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: '2-digit' });
  }

  // Handle click with modifiers
  function handleClick(e) {
    // Shift+Click: range selection
    if (e.shiftKey) {
      e.preventDefault();
      selectRange(index);
      return;
    }

    // Ctrl/Cmd+Click: toggle this email
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      toggleSelection(email.id, index);
      return;
    }

    // Normal click: update lastSelectedIndex and dispatch to parent
    lastSelectedIndex.set(index);
    dispatch('click');
  }

  // Handle checkbox click
  function handleCheckboxClick(e) {
    e.stopPropagation();
    toggleSelection(email.id, index);
  }

  // Handle double click to open
  function handleDoubleClick() {
    dispatch('open');
  }

  // Drag selection handlers
  function handleMouseDown(e) {
    // Only start drag selection on left click without modifiers
    if (e.button === 0 && !e.ctrlKey && !e.metaKey && !e.shiftKey) {
      // Check if click is in the left area (checkbox zone)
      var rect = e.currentTarget.getBoundingClientRect();
      var leftZone = rect.left + 50; // Checkbox area width

      if (e.clientX < leftZone) {
        e.preventDefault();
        startDragSelection(index);
      }
    }
  }

  function handleMouseEnter() {
    hovering = true;
    if ($dragSelecting) {
      updateDragSelection(index);
    }
  }

  function handleMouseUp() {
    if ($dragSelecting) {
      endDragSelection();
    }
  }
</script>

<svelte:window on:mouseup={handleMouseUp} />

<div
  class="email-row"
  class:selected
  class:checked={isChecked}
  class:unread={!email.isRead}
  class:drag-selecting={$dragSelecting}
  role="button"
  tabindex="0"
  on:click={handleClick}
  on:dblclick={handleDoubleClick}
  on:keydown={(e) => e.key === 'Enter' && handleClick(e)}
  on:mousedown={handleMouseDown}
  on:mouseenter={handleMouseEnter}
  on:mouseleave={() => hovering = false}
>
  <!-- Checkbox / Star area -->
  <div class="checkbox-area">
    {#if showCheckbox}
      <button
        class="checkbox"
        class:checked={isChecked}
        on:click={handleCheckboxClick}
        transition:scale={{ duration: 150 }}
      >
        {#if isChecked}
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/>
          </svg>
        {/if}
      </button>
    {:else}
      <div class="star-area">
        {#if email.isStarred}
          <span class="star" title="Starred">★</span>
        {:else}
          <span class="star empty">☆</span>
        {/if}
      </div>
    {/if}
  </div>

  <div class="from truncate">
    {email.fromName || email.fromEmail}
  </div>

  <div class="content">
    <span class="subject truncate">{email.subject || '(sem assunto)'}</span>
    <span class="separator"> - </span>
    <span class="snippet truncate">{email.snippet}</span>
  </div>

  <div class="meta">
    {#if email.threadCount > 1}
      <span class="thread-count" title="{email.threadCount} messages in thread">[{email.threadCount}]</span>
    {/if}
    {#if email.hasAttachments}
      <span class="attachment" title="Has attachments">📎</span>
    {/if}
    <span class="date">{formatDate(email.date)}</span>
  </div>

  <!-- Selection indicator line -->
  {#if isChecked}
    <div class="selection-indicator"></div>
  {/if}
</div>

<style>
  .email-row {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    border-bottom: 1px solid var(--border-subtle);
    cursor: pointer;
    transition: background var(--transition-fast);
    position: relative;
  }

  .email-row:hover {
    background: var(--bg-hover);
  }

  .email-row.selected {
    background: var(--bg-selected);
  }

  .email-row.checked {
    background: var(--bg-selected);
  }

  .email-row.checked.selected {
    background: var(--bg-selected);
  }

  .email-row.drag-selecting {
    cursor: crosshair;
    user-select: none;
  }

  .email-row.unread {
    font-weight: 600;
  }

  .email-row.unread .from,
  .email-row.unread .subject {
    color: var(--text-primary);
  }

  /* Selection indicator - vertical line on left */
  .selection-indicator {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 3px;
    background: var(--accent-primary);
    border-radius: 0 2px 2px 0;
  }

  /* Checkbox area */
  .checkbox-area {
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .checkbox {
    width: 18px;
    height: 18px;
    border: 2px solid var(--border-color);
    border-radius: 4px;
    background: transparent;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s ease;
    padding: 0;
  }

  .checkbox:hover {
    border-color: var(--accent-primary);
    background: var(--bg-hover);
  }

  .checkbox.checked {
    border-color: var(--accent-primary);
    background: var(--accent-primary);
  }

  .checkbox svg {
    width: 14px;
    height: 14px;
    color: white;
  }

  .star-area {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    text-align: center;
  }

  .star {
    color: var(--accent-warning);
    font-size: var(--font-md);
  }

  .star.empty {
    color: var(--text-muted);
    opacity: 0.3;
  }

  .from {
    flex-shrink: 0;
    width: 180px;
    color: var(--text-secondary);
    font-size: var(--font-sm);
  }

  .content {
    flex: 1;
    display: flex;
    align-items: center;
    min-width: 0;
    overflow: hidden;
  }

  .subject {
    color: var(--text-primary);
    font-size: var(--font-sm);
  }

  .separator {
    color: var(--text-muted);
    flex-shrink: 0;
    margin: 0 var(--space-xs);
  }

  .snippet {
    color: var(--text-muted);
    font-size: var(--font-sm);
    font-weight: 400;
  }

  .meta {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    flex-shrink: 0;
  }

  .attachment {
    font-size: var(--font-sm);
  }

  .date {
    color: var(--text-muted);
    font-size: var(--font-xs);
    min-width: 60px;
    text-align: right;
  }

  .thread-count {
    font-size: var(--font-xs);
    font-weight: 600;
    color: var(--accent-primary);
    background: var(--bg-secondary);
    padding: 1px 6px;
    border-radius: 8px;
    border: 1px solid var(--accent-primary);
  }

  .truncate {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
