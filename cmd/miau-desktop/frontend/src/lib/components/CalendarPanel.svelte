<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import {
    currentWeekStart,
    calendarEvents,
    calendarLoading,
    eventsByDay,
    weekLabel,
    isCurrentWeek,
    eventTypeColors,
    eventTypeLabels,
    loadWeekEvents,
    previousWeek,
    nextWeek,
    goToToday,
    deleteEvent,
    toggleEventComplete,
    showCalendarPanel,
    openCreateEventModal,
    openEditEventModal,
    googleCalendarConnected,
    googleSyncLoading,
    checkGoogleCalendarConnection,
    syncFromGoogle
  } from '../stores/calendar.js';

  var dispatch = createEventDispatcher();
  var syncResult = null;

  onMount(async () => {
    loadWeekEvents();
    var connected = await checkGoogleCalendarConnection();
    console.log('[CalendarPanel] Google Calendar connected:', connected);

    // Auto-sync on first load if connected
    if (connected) {
      console.log('[CalendarPanel] Auto-syncing from Google Calendar...');
      handleGoogleSync();
    } else {
      console.log('[CalendarPanel] Google Calendar not connected. Run "miau auth" and restart the app.');
    }
  });

  async function handleGoogleSync() {
    syncResult = null;
    try {
      var count = await syncFromGoogle();
      syncResult = { success: true, count };
      setTimeout(() => syncResult = null, 5000);
    } catch (err) {
      console.error('[CalendarPanel] Sync error:', err);
      var errorMsg = err.message || err || 'Sync failed';
      // Check for common errors
      if (errorMsg.includes('403') || errorMsg.includes('insufficient')) {
        errorMsg = 'Permission denied. Delete ~/.config/miau/tokens/*.json and run "miau auth" again';
      }
      syncResult = { success: false, error: errorMsg };
    }
  }

  function closePanel() {
    showCalendarPanel.set(false);
  }

  function getEventColor(event) {
    return event.color || eventTypeColors[event.eventType] || eventTypeColors.custom;
  }

  function formatEventTime(event) {
    if (event.allDay) return 'All day';
    var time = new Date(event.startTime);
    return time.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
  }

  async function handleToggleComplete(event) {
    try {
      await toggleEventComplete(event.id);
    } catch (err) {
      console.error('Failed to toggle event:', err);
    }
  }

  async function handleDelete(event) {
    if (confirm(`Delete "${event.title}"?`)) {
      try {
        await deleteEvent(event.id);
      } catch (err) {
        console.error('Failed to delete event:', err);
      }
    }
  }

  function handleCreateEvent(day) {
    openCreateEventModal(day.date);
  }
</script>

<div class="calendar-panel">
  <!-- Header -->
  <div class="panel-header">
    <button class="back-btn" on:click={closePanel}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 18l-6-6 6-6"/>
      </svg>
      <span>Back</span>
    </button>

    <div class="header-center">
      <button class="nav-btn" on:click={previousWeek}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M15 18l-6-6 6-6"/>
        </svg>
      </button>
      <h2 class="week-title">{$weekLabel}</h2>
      <button class="nav-btn" on:click={nextWeek}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 18l6-6-6-6"/>
        </svg>
      </button>
    </div>

    <div class="header-actions">
      {#if $googleCalendarConnected}
        <button
          class="sync-btn"
          class:loading={$googleSyncLoading}
          on:click={handleGoogleSync}
          disabled={$googleSyncLoading}
          title="Sync from Google Calendar"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class:spinning={$googleSyncLoading}>
            <path d="M21 12a9 9 0 11-9-9"/>
            <path d="M21 3v6h-6"/>
          </svg>
          {#if $googleSyncLoading}
            <span>Syncing...</span>
          {:else}
            <span>Google Sync</span>
          {/if}
        </button>
      {:else}
        <span class="not-connected" title="Run 'miau auth' and restart the app to connect Google Calendar">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          Google Calendar not connected
        </span>
      {/if}
      {#if syncResult}
        <span class="sync-result" class:success={syncResult.success} class:error={!syncResult.success}>
          {#if syncResult.success}
            Synced {syncResult.count} events
          {:else}
            {syncResult.error}
          {/if}
        </span>
      {/if}
      {#if !$isCurrentWeek}
        <button class="today-btn" on:click={goToToday}>Today</button>
      {/if}
      <button class="add-btn" on:click={() => openCreateEventModal()}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 5v14M5 12h14"/>
        </svg>
        <span>Add Event</span>
      </button>
    </div>
  </div>

  <!-- Week Grid -->
  <div class="week-grid">
    {#if $calendarLoading}
      <div class="loading">
        <span>Loading events...</span>
      </div>
    {:else}
      {#each $eventsByDay as day}
        <div class="day-column" class:today={day.isToday}>
          <div class="day-header">
            <span class="day-name">{day.dayName}</span>
            <span class="day-date" class:today={day.isToday}>{day.date.getDate()}</span>
          </div>

          <div class="day-content">
            {#if day.events.length === 0}
              <button class="empty-day" on:click={() => handleCreateEvent(day)}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 5v14M5 12h14"/>
                </svg>
              </button>
            {:else}
              {#each day.events as event}
                <div
                  class="event-card"
                  class:completed={event.isCompleted}
                  style="--event-color: {getEventColor(event)}"
                >
                  <div class="event-color-bar"></div>
                  <div class="event-content">
                    <div class="event-header">
                      <span class="event-time">{formatEventTime(event)}</span>
                      <span class="event-type">{eventTypeLabels[event.eventType] || 'Event'}</span>
                    </div>
                    <button class="event-title" on:click={() => openEditEventModal(event)}>
                      {event.title}
                    </button>
                    {#if event.description}
                      <p class="event-description">{event.description}</p>
                    {/if}
                  </div>
                  <div class="event-actions">
                    <button
                      class="action-btn complete"
                      class:is-completed={event.isCompleted}
                      on:click={() => handleToggleComplete(event)}
                      title={event.isCompleted ? 'Mark incomplete' : 'Mark complete'}
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M20 6L9 17l-5-5"/>
                      </svg>
                    </button>
                    <button
                      class="action-btn delete"
                      on:click={() => handleDelete(event)}
                      title="Delete"
                    >
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M18 6L6 18M6 6l12 12"/>
                      </svg>
                    </button>
                  </div>
                </div>
              {/each}
              <button class="add-event-day" on:click={() => handleCreateEvent(day)}>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 5v14M5 12h14"/>
                </svg>
                <span>Add</span>
              </button>
            {/if}
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .calendar-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-secondary);
  }

  .back-btn {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .back-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
  }

  .header-center {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .week-title {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    min-width: 200px;
    text-align: center;
  }

  .nav-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .nav-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .today-btn {
    padding: var(--space-xs) var(--space-sm);
    border: 1px solid var(--border-color);
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--font-sm);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .today-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
    border-color: var(--accent-primary);
  }

  .add-btn {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-md);
    border: none;
    background: var(--accent-primary);
    color: var(--bg-primary);
    font-size: var(--font-sm);
    font-weight: 500;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .add-btn:hover {
    opacity: 0.9;
  }

  .sync-btn {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs) var(--space-sm);
    border: 1px solid var(--border-color);
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--font-sm);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .sync-btn:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--accent-primary);
    border-color: var(--accent-primary);
  }

  .sync-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .sync-btn.loading {
    color: var(--accent-primary);
  }

  .sync-btn svg.spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .sync-result {
    font-size: var(--font-xs);
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
  }

  .sync-result.success {
    color: var(--accent-success);
    background: rgba(var(--accent-success-rgb), 0.1);
  }

  .sync-result.error {
    color: var(--accent-error);
    background: rgba(var(--accent-error-rgb), 0.1);
  }

  .not-connected {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: var(--font-xs);
    color: var(--text-muted);
    padding: var(--space-xs) var(--space-sm);
    cursor: help;
  }

  .week-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    flex: 1;
    overflow: hidden;
  }

  .day-column {
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--border-color);
    min-height: 0;
  }

  .day-column:last-child {
    border-right: none;
  }

  .day-column.today {
    background: rgba(var(--accent-primary-rgb), 0.05);
  }

  .day-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: var(--space-sm);
    border-bottom: 1px solid var(--border-color);
    background: var(--bg-secondary);
  }

  .day-name {
    font-size: var(--font-xs);
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .day-date {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
  }

  .day-date.today {
    background: var(--accent-primary);
    color: var(--bg-primary);
  }

  .day-content {
    flex: 1;
    padding: var(--space-xs);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .empty-day {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    opacity: 0.3;
    transition: all var(--transition-fast);
  }

  .empty-day:hover {
    opacity: 1;
    color: var(--accent-primary);
  }

  .event-card {
    display: flex;
    background: var(--bg-secondary);
    border-radius: var(--radius-sm);
    overflow: hidden;
    transition: all var(--transition-fast);
  }

  .event-card:hover {
    background: var(--bg-tertiary);
  }

  .event-card.completed {
    opacity: 0.6;
  }

  .event-color-bar {
    width: 4px;
    background: var(--event-color);
    flex-shrink: 0;
  }

  .event-content {
    flex: 1;
    padding: var(--space-xs) var(--space-sm);
    min-width: 0;
  }

  .event-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-xs);
    margin-bottom: 2px;
  }

  .event-time {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .event-type {
    font-size: 9px;
    color: var(--event-color);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .event-title {
    display: block;
    font-size: var(--font-sm);
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 0;
    text-align: left;
    width: 100%;
  }

  .event-card.completed .event-title {
    text-decoration: line-through;
    color: var(--text-muted);
  }

  .event-title:hover {
    color: var(--accent-primary);
  }

  .event-description {
    font-size: var(--font-xs);
    color: var(--text-muted);
    margin: 2px 0 0 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .event-actions {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--space-xs);
    opacity: 0;
    transition: opacity var(--transition-fast);
  }

  .event-card:hover .event-actions {
    opacity: 1;
  }

  .action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .action-btn:hover {
    background: var(--bg-hover);
  }

  .action-btn.complete:hover,
  .action-btn.complete.is-completed {
    color: var(--accent-success);
  }

  .action-btn.delete:hover {
    color: var(--accent-error);
  }

  .add-event-day {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    padding: var(--space-xs);
    border: 1px dashed var(--border-color);
    background: transparent;
    color: var(--text-muted);
    font-size: var(--font-xs);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .add-event-day:hover {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
  }

  .loading {
    grid-column: span 7;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-xl);
    color: var(--text-muted);
  }
</style>
