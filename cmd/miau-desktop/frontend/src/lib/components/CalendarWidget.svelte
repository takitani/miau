<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import {
    currentWeekStart,
    calendarEvents,
    calendarLoading,
    eventCounts,
    eventsByDay,
    weekLabel,
    isCurrentWeek,
    eventTypeColors,
    loadWeekEvents,
    previousWeek,
    nextWeek,
    goToToday,
    showCalendarPanel
  } from '../stores/calendar.js';

  var dispatch = createEventDispatcher();

  onMount(() => {
    loadWeekEvents();
  });

  function handleEventClick(event) {
    dispatch('eventClick', event);
  }

  function handleDayClick(day) {
    dispatch('dayClick', day);
  }

  function openFullCalendar() {
    showCalendarPanel.set(true);
  }

  function getEventColor(event) {
    return event.color || eventTypeColors[event.eventType] || eventTypeColors.custom;
  }
</script>

<div class="calendar-widget">
  <!-- Header with navigation -->
  <div class="widget-header">
    <div class="nav-buttons">
      <button class="nav-btn" on:click={previousWeek} title="Previous week">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M15 18l-6-6 6-6"/>
        </svg>
      </button>
      <button class="nav-btn" on:click={nextWeek} title="Next week">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 18l6-6-6-6"/>
        </svg>
      </button>
    </div>
    <span class="week-label">{$weekLabel}</span>
    {#if !$isCurrentWeek}
      <button class="today-btn" on:click={goToToday} title="Go to today">
        Today
      </button>
    {/if}
  </div>

  <!-- Week view -->
  <div class="week-view">
    {#if $calendarLoading}
      <div class="loading">Loading...</div>
    {:else}
      <div class="days-grid">
        {#each $eventsByDay as day}
          <button
            class="day-column"
            class:today={day.isToday}
            on:click={() => handleDayClick(day)}
          >
            <div class="day-header">
              <span class="day-name">{day.dayName}</span>
              <span class="day-date" class:today={day.isToday}>{day.date.getDate()}</span>
            </div>
            <div class="day-events">
              {#each day.events.slice(0, 3) as event}
                <div
                  class="event-dot"
                  class:completed={event.isCompleted}
                  style="--event-color: {getEventColor(event)}"
                  title={event.title}
                  on:click|stopPropagation={() => handleEventClick(event)}
                ></div>
              {/each}
              {#if day.events.length > 3}
                <span class="more-events">+{day.events.length - 3}</span>
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Upcoming events preview -->
  {#if !$calendarLoading && $calendarEvents.length > 0}
    <div class="upcoming-section">
      <div class="section-title">Upcoming</div>
      <div class="upcoming-list">
        {#each $calendarEvents.filter(e => !e.isCompleted).slice(0, 3) as event}
          <button
            class="upcoming-item"
            on:click={() => handleEventClick(event)}
          >
            <div
              class="event-indicator"
              style="background: {getEventColor(event)}"
            ></div>
            <div class="event-info">
              <span class="event-title">{event.title}</span>
              <span class="event-date">
                {new Date(event.startTime).toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit' })}
              </span>
            </div>
          </button>
        {/each}
      </div>
    </div>
  {:else if !$calendarLoading}
    <div class="empty-state">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
        <line x1="16" y1="2" x2="16" y2="6"/>
        <line x1="8" y1="2" x2="8" y2="6"/>
        <line x1="3" y1="10" x2="21" y2="10"/>
      </svg>
      <span>No events</span>
    </div>
  {/if}

  <!-- Footer with expand button -->
  <button class="expand-btn" on:click={openFullCalendar}>
    <span>View full calendar</span>
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M5 12h14M12 5l7 7-7 7"/>
    </svg>
  </button>
</div>

<style>
  .calendar-widget {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }

  .widget-header {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: 0 var(--space-xs);
  }

  .nav-buttons {
    display: flex;
    gap: 2px;
  }

  .nav-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
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

  .week-label {
    flex: 1;
    font-size: var(--font-xs);
    color: var(--text-secondary);
    text-align: center;
  }

  .today-btn {
    padding: 2px 8px;
    border: none;
    background: var(--bg-tertiary);
    color: var(--accent-primary);
    font-size: var(--font-xs);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .today-btn:hover {
    background: var(--bg-hover);
  }

  .week-view {
    padding: 0 var(--space-xs);
  }

  .days-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 2px;
  }

  .day-column {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: var(--space-xs);
    background: transparent;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .day-column:hover {
    background: var(--bg-hover);
  }

  .day-column.today {
    background: var(--bg-tertiary);
  }

  .day-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
  }

  .day-name {
    font-size: 9px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .day-date {
    font-size: var(--font-sm);
    font-weight: 500;
    color: var(--text-primary);
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
  }

  .day-date.today {
    background: var(--accent-primary);
    color: var(--bg-primary);
  }

  .day-events {
    display: flex;
    gap: 2px;
    flex-wrap: wrap;
    justify-content: center;
    min-height: 14px;
  }

  .event-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--event-color);
    cursor: pointer;
    transition: transform var(--transition-fast);
  }

  .event-dot:hover {
    transform: scale(1.3);
  }

  .event-dot.completed {
    opacity: 0.4;
  }

  .more-events {
    font-size: 9px;
    color: var(--text-muted);
  }

  .upcoming-section {
    padding: var(--space-xs);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }

  .section-title {
    font-size: var(--font-xs);
    color: var(--text-muted);
    margin-bottom: var(--space-xs);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .upcoming-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .upcoming-item {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs);
    background: transparent;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    width: 100%;
    text-align: left;
    transition: all var(--transition-fast);
  }

  .upcoming-item:hover {
    background: var(--bg-hover);
  }

  .event-indicator {
    width: 3px;
    height: 20px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .event-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .event-title {
    font-size: var(--font-sm);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .event-date {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-md);
    color: var(--text-muted);
    font-size: var(--font-sm);
  }

  .empty-state svg {
    opacity: 0.5;
  }

  .expand-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-xs);
    padding: var(--space-xs);
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: var(--font-xs);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .expand-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
  }

  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-md);
    color: var(--text-muted);
    font-size: var(--font-sm);
  }
</style>
