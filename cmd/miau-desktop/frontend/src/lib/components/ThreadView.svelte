<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import ThreadMessage from './ThreadMessage.svelte';
  import ThreadMinimap from './ThreadMinimap.svelte';

  export var emailId;

  var dispatch = createEventDispatcher();

  var thread = null;
  var loading = true;
  var error = null;
  var selectedIndex = 0;
  var expandedIndices = [0]; // First message expanded by default (array for reactivity)
  var showMinimap = true;
  var scrollContainer;
  var scrollProgress = 0;

  // Check if index is expanded
  function isExpanded(index) {
    return expandedIndices.includes(index);
  }

  // Wails backend access
  var App = window.go?.desktop?.App;

  // Color palette for participants
  var COLORS = [
    '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', '#FFEAA7',
    '#DDA0DD', '#98D8C8', '#F7DC6F', '#BB8FCE', '#85C1E9',
    '#F8B500', '#00CED1', '#FF69B4', '#32CD32', '#FFD700'
  ];

  // Generate participant colors
  $: participantColors = (() => {
    if (!thread?.participants) return {};
    var colors = {};
    thread.participants.forEach((email, i) => {
      colors[email] = COLORS[i % COLORS.length];
    });
    return colors;
  })();

  // Get color for a message
  function getMessageColor(msg) {
    return participantColors[msg.fromEmail] || '#666';
  }

  // Load thread data
  async function loadThread() {
    loading = true;
    error = null;
    try {
      if (!App) {
        App = window.go?.desktop?.App;
      }
      if (!App) {
        error = 'Backend not available';
        return;
      }
      thread = await App.GetThread(emailId);
      if (!thread) {
        error = 'Thread not found';
      } else {
        // Expand first (newest) message
        expandedIndices = [0];
        selectedIndex = 0;
      }
    } catch (e) {
      error = e.message || 'Failed to load thread';
    } finally {
      loading = false;
    }
  }

  // Navigate to message
  function navigateToMessage(index) {
    selectedIndex = index;
    scrollToMessage(index);
  }

  // Scroll to specific message
  function scrollToMessage(index) {
    if (!scrollContainer) return;
    var messages = scrollContainer.querySelectorAll('.message-wrapper');
    if (messages[index]) {
      messages[index].scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }

  // Handle scroll for minimap sync
  function handleScroll() {
    if (!scrollContainer) return;
    var { scrollTop, scrollHeight, clientHeight } = scrollContainer;
    scrollProgress = scrollTop / Math.max(1, scrollHeight - clientHeight);
  }

  // Toggle message expansion
  function toggleMessage(index) {
    if (expandedIndices.includes(index)) {
      expandedIndices = expandedIndices.filter(i => i !== index);
    } else {
      expandedIndices = [...expandedIndices, index];
    }
  }

  // Select message
  function selectMessage(index) {
    selectedIndex = index;
  }

  // Collapse all except current
  function collapseAll() {
    expandedIndices = [selectedIndex];
  }

  // Expand all messages
  function expandAll() {
    expandedIndices = thread.messages.map((_, i) => i);
  }

  // Mark thread as read
  async function markAsRead() {
    if (!thread || !App) return;
    try {
      await App.MarkThreadAsRead(thread.threadId);
      thread.messages = thread.messages.map(m => ({ ...m, isRead: true }));
      thread.isRead = true;
    } catch (e) {
      console.error('Failed to mark as read:', e);
    }
  }

  // Mark thread as unread
  async function markAsUnread() {
    if (!thread || !App) return;
    try {
      await App.MarkThreadAsUnread(thread.threadId);
      // Mark first (newest) as unread
      if (thread.messages.length > 0) {
        thread.messages[0] = { ...thread.messages[0], isRead: false };
      }
      thread.isRead = false;
    } catch (e) {
      console.error('Failed to mark as unread:', e);
    }
  }

  // Close thread view
  function close() {
    dispatch('close');
  }

  // Keyboard navigation
  function handleKeydown(e) {
    if (!thread) return;

    switch (e.key) {
      case 'Escape':
        close();
        break;
      case 'j':
      case 'ArrowDown':
        if (selectedIndex < thread.messages.length - 1) {
          selectedIndex++;
          scrollToMessage(selectedIndex);
        }
        e.preventDefault();
        break;
      case 'k':
      case 'ArrowUp':
        if (selectedIndex > 0) {
          selectedIndex--;
          scrollToMessage(selectedIndex);
        }
        e.preventDefault();
        break;
      case 'Enter':
      case ' ':
        toggleMessage(selectedIndex);
        e.preventDefault();
        break;
      case 'm':
        showMinimap = !showMinimap;
        e.preventDefault();
        break;
      case 't':
        collapseAll();
        e.preventDefault();
        break;
      case 'T':
        expandAll();
        e.preventDefault();
        break;
      case 'r':
        if (e.shiftKey) {
          markAsUnread();
        } else {
          markAsRead();
        }
        e.preventDefault();
        break;
    }
  }

  // Track current emailId to detect changes
  var currentEmailId = null;

  // Reload when emailId changes (reactive statement)
  $: if (emailId && emailId !== currentEmailId) {
    currentEmailId = emailId;
    loadThread();
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="thread-view">
  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Carregando thread...</p>
    </div>
  {:else if error}
    <div class="error-state">
      <span class="error-icon">❌</span>
      <p>{error}</p>
      <button class="btn-retry" on:click={loadThread}>Tentar novamente</button>
      <button class="btn-close" on:click={close}>Voltar</button>
    </div>
  {:else if thread}
    <!-- Thread Header -->
    <header class="thread-header">
      <button class="btn-back" on:click={close} title="Voltar (Esc)">
        ← Voltar
      </button>

      <div class="thread-title">
        <h2>{thread.subject}</h2>
        <div class="thread-meta">
          <span class="message-count">
            {thread.messageCount} {thread.messageCount === 1 ? 'mensagem' : 'mensagens'}
          </span>
          {#if !thread.isRead}
            <span class="unread-badge">não lida</span>
          {/if}
        </div>
      </div>

      <div class="thread-actions">
        <button
          class="btn-action"
          on:click={() => showMinimap = !showMinimap}
          title="Toggle minimap (m)"
          class:active={showMinimap}
        >
          {showMinimap ? '◧' : '▣'}
        </button>
        <button class="btn-action" on:click={collapseAll} title="Colapsar todas (t)">
          ⊟
        </button>
        <button class="btn-action" on:click={expandAll} title="Expandir todas (T)">
          ⊞
        </button>
        <button class="btn-action" on:click={markAsRead} title="Marcar como lida (r)">
          ✓
        </button>
      </div>
    </header>

    <!-- Participants bar -->
    <div class="participants-bar">
      {#each thread.participants as email}
        {@const color = participantColors[email]}
        <span class="participant-chip" style="--chip-color: {color}">
          <span class="chip-dot" style="background: {color}"></span>
          {email.split('@')[0]}
        </span>
      {/each}
    </div>

    <!-- Main content area -->
    <div class="thread-content">
      <!-- Messages list -->
      <div
        class="messages-container"
        bind:this={scrollContainer}
        on:scroll={handleScroll}
      >
        {#each thread.messages as msg, i}
          <div class="message-wrapper">
            <ThreadMessage
              message={msg}
              isExpanded={expandedIndices.includes(i)}
              isSelected={i === selectedIndex}
              participantColor={getMessageColor(msg)}
              on:toggle={() => toggleMessage(i)}
              on:select={() => selectMessage(i)}
            />
          </div>
        {/each}
      </div>

      <!-- Minimap -->
      {#if showMinimap}
        <ThreadMinimap
          messages={thread.messages}
          {selectedIndex}
          {participantColors}
          {scrollProgress}
          on:navigate={(e) => navigateToMessage(e.detail.index)}
        />
      {/if}
    </div>

    <!-- Footer with keyboard hints -->
    <footer class="thread-footer">
      <span class="hint">↑↓ navegar</span>
      <span class="hint">Enter expandir</span>
      <span class="hint">m minimap</span>
      <span class="hint">t colapsar</span>
      <span class="hint">r marcar lida</span>
      <span class="hint">Esc voltar</span>
    </footer>
  {/if}
</div>

<style>
  .thread-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
  }

  /* Loading & Error States */
  .loading-state,
  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    gap: 16px;
    color: var(--text-muted);
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-icon {
    font-size: 48px;
  }

  .btn-retry,
  .btn-close {
    padding: 8px 16px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
  }

  .btn-retry {
    background: var(--accent);
    color: white;
  }

  .btn-close {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  /* Thread Header */
  .thread-header {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 16px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
  }

  .btn-back {
    padding: 6px 12px;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    cursor: pointer;
    font-size: 13px;
    transition: all 0.15s;
  }

  .btn-back:hover {
    background: var(--bg-hover);
    border-color: var(--accent);
  }

  .thread-title {
    flex: 1;
    min-width: 0;
  }

  .thread-title h2 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .thread-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 4px;
  }

  .message-count {
    font-size: 12px;
    color: var(--text-muted);
  }

  .unread-badge {
    font-size: 10px;
    padding: 2px 6px;
    background: var(--accent);
    color: white;
    border-radius: 10px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .thread-actions {
    display: flex;
    gap: 4px;
  }

  .btn-action {
    width: 32px;
    height: 32px;
    border: 1px solid var(--border);
    background: var(--bg-primary);
    border-radius: 6px;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }

  .btn-action:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
    border-color: var(--accent);
  }

  .btn-action.active {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
  }

  /* Participants Bar */
  .participants-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    padding: 8px 16px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
  }

  .participant-chip {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    background: var(--bg-primary);
    border-radius: 16px;
    font-size: 12px;
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .chip-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  /* Main Content */
  .thread-content {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .messages-container {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    scroll-behavior: smooth;
  }

  .message-wrapper {
    margin-bottom: 4px;
  }

  /* Footer */
  .thread-footer {
    display: flex;
    justify-content: center;
    gap: 16px;
    padding: 8px 16px;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border);
  }

  .hint {
    font-size: 11px;
    color: var(--text-muted);
  }

  .hint::before {
    content: '';
    display: inline-block;
    width: 4px;
    height: 4px;
    background: var(--accent);
    border-radius: 50%;
    margin-right: 6px;
    vertical-align: middle;
  }
</style>
