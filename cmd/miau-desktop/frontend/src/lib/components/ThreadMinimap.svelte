<script>
  import { createEventDispatcher } from 'svelte';

  export var messages = [];
  export var selectedIndex = 0;
  export var participantColors = {};
  export var scrollProgress = 0; // 0-1 representing scroll position

  var dispatch = createEventDispatcher();

  // High-contrast color palette for participants
  var distinctColors = [
    '#e74c3c', // Red
    '#3498db', // Blue
    '#2ecc71', // Green
    '#9b59b6', // Purple
    '#f39c12', // Orange
    '#1abc9c', // Teal
    '#e91e63', // Pink
    '#00bcd4', // Cyan
    '#ff5722', // Deep Orange
    '#8bc34a', // Light Green
  ];

  // Generate distinct colors for participants
  function getDistinctColor(email, index) {
    if (participantColors[email]) return participantColors[email];
    return distinctColors[index % distinctColors.length];
  }

  // Build a map of emails to distinct colors
  $: participantList = [...new Set(messages.map(m => m.fromEmail))];
  $: colorMap = participantList.reduce((acc, email, i) => {
    acc[email] = distinctColors[i % distinctColors.length];
    return acc;
  }, {});

  // Calculate dot positions with even spacing
  $: dotPositions = messages.map((_, i) => {
    return (i / Math.max(1, messages.length - 1)) * 100;
  });

  function handleDotClick(index) {
    dispatch('navigate', { index });
  }

  function handleTrackClick(e) {
    var rect = e.currentTarget.getBoundingClientRect();
    var y = e.clientY - rect.top;
    var percentage = y / rect.height;
    var index = Math.round(percentage * (messages.length - 1));
    index = Math.max(0, Math.min(messages.length - 1, index));
    dispatch('navigate', { index });
  }

  // Get participant initials (2 letters)
  function getInitials(name) {
    if (!name) return '??';
    var parts = name.trim().split(/\s+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
    }
    return name.slice(0, 2).toUpperCase();
  }
</script>

<aside class="minimap">
  <!-- Header with count -->
  <div class="minimap-header">
    <span class="count">{messages.length}</span>
    <span class="label">msgs</span>
  </div>

  <!-- Scrollable track with dots -->
  <div
    class="minimap-track"
    on:click={handleTrackClick}
    role="slider"
    aria-label="Thread navigation"
    aria-valuemin="0"
    aria-valuemax={messages.length - 1}
    aria-valuenow={selectedIndex}
    tabindex="0"
  >
    <!-- Scroll indicator -->
    <div class="scroll-indicator" style="top: {scrollProgress * 100}%"></div>

    <!-- Connection line -->
    <div class="connection-line"></div>

    <!-- Dots -->
    {#each messages as msg, i}
      {@const color = colorMap[msg.fromEmail] || participantColors[msg.fromEmail] || '#666'}
      {@const isSelected = i === selectedIndex}
      {@const isUnread = !msg.isRead}
      <button
        class="dot"
        class:selected={isSelected}
        class:unread={isUnread}
        style="
          top: {dotPositions[i]}%;
          --dot-color: {color};
        "
        on:click|stopPropagation={() => handleDotClick(i)}
        title="{msg.fromName || msg.fromEmail} - {new Date(msg.date).toLocaleString('pt-BR')}"
      >
        <span class="dot-inner" class:selected={isSelected} class:pulse={isUnread && !isSelected}>
          {getInitials(msg.fromName)}
        </span>
      </button>
    {/each}
  </div>

  <!-- Participant legend -->
  {#if participantList.length > 0}
    <div class="legend">
      {#each participantList as email, i}
        {@const color = colorMap[email] || '#666'}
        {@const name = messages.find(m => m.fromEmail === email)?.fromName || email}
        <div class="legend-item" title={email}>
          <span class="legend-dot" style="background: {color}">
            {getInitials(name)}
          </span>
          <span class="legend-name">
            {name.split(' ')[0].slice(0, 8)}
          </span>
        </div>
      {/each}
    </div>
  {/if}
</aside>

<style>
  .minimap {
    width: 72px;
    background: var(--bg-tertiary, #1a1a2e);
    border-left: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    padding: 12px 0;
    user-select: none;
  }

  .minimap-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding-bottom: 12px;
    border-bottom: 1px solid var(--border);
    margin: 0 8px 12px;
  }

  .count {
    font-size: 20px;
    font-weight: 700;
    color: var(--accent);
    line-height: 1;
  }

  .label {
    font-size: 10px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .minimap-track {
    flex: 1;
    position: relative;
    margin: 0 8px;
    cursor: pointer;
    min-height: 100px;
    padding: 16px 0; /* Space for dots at edges */
  }

  .connection-line {
    position: absolute;
    left: 50%;
    top: 0;
    bottom: 0;
    width: 2px;
    background: linear-gradient(
      to bottom,
      var(--border) 0%,
      var(--accent) 50%,
      var(--border) 100%
    );
    opacity: 0.3;
    transform: translateX(-50%);
  }

  .scroll-indicator {
    position: absolute;
    left: 0;
    right: 0;
    height: 3px;
    background: var(--accent);
    opacity: 0.6;
    border-radius: 2px;
    transition: top 0.1s ease;
    z-index: 1;
  }

  .dot {
    position: absolute;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2;
    transition: transform 0.15s ease;
  }

  .dot:hover {
    transform: translate(-50%, -50%) scale(1.2);
  }

  .dot-inner {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: var(--dot-color);
    transition: all 0.15s ease;
    box-shadow: 0 0 0 2px var(--bg-tertiary, #1a1a2e);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 9px;
    font-weight: 700;
    color: white;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  }

  .dot-inner.selected {
    width: 28px;
    height: 28px;
    font-size: 10px;
    box-shadow: 0 0 0 3px var(--accent), 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .dot-inner.pulse {
    animation: pulse 2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% {
      box-shadow: 0 0 0 2px var(--bg-tertiary, #1a1a2e);
    }
    50% {
      box-shadow: 0 0 0 2px var(--bg-tertiary, #1a1a2e),
                  0 0 0 6px color-mix(in srgb, var(--dot-color) 30%, transparent);
    }
  }

  .legend {
    margin-top: 16px;
    padding: 12px 6px 0;
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex-shrink: 0;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .legend-dot {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 8px;
    font-weight: 700;
    color: white;
    text-shadow: 0 1px 1px rgba(0, 0, 0, 0.3);
  }

  .legend-name {
    font-size: 10px;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
