<script>
  import { onMount } from 'svelte';
  import {
    selectedEvent,
    showEventModal,
    eventTypeColors,
    eventTypeLabels,
    createEvent,
    updateEvent,
    deleteEvent,
    closeEventModal
  } from '../stores/calendar.js';

  var saving = false;
  var deleting = false;
  var error = null;

  // Form state
  var title = '';
  var description = '';
  var eventType = 'custom';
  var startDate = '';
  var startTime = '';
  var endDate = '';
  var endTime = '';
  var allDay = false;
  var color = eventTypeColors.custom;

  // Subscribe to selected event
  $: if ($selectedEvent) {
    initForm($selectedEvent);
  }

  function initForm(event) {
    title = event.title || '';
    description = event.description || '';
    eventType = event.eventType || 'custom';
    allDay = event.allDay || false;
    color = event.color || eventTypeColors[eventType] || eventTypeColors.custom;

    var start = event.startTime ? new Date(event.startTime) : new Date();
    startDate = formatDateInput(start);
    startTime = formatTimeInput(start);

    if (event.endTime) {
      var end = new Date(event.endTime);
      endDate = formatDateInput(end);
      endTime = formatTimeInput(end);
    } else {
      endDate = startDate;
      endTime = '';
    }
  }

  function formatDateInput(date) {
    return date.toISOString().split('T')[0];
  }

  function formatTimeInput(date) {
    return date.toTimeString().slice(0, 5);
  }

  function handleEventTypeChange() {
    color = eventTypeColors[eventType] || eventTypeColors.custom;
  }

  async function handleSave() {
    if (!title.trim()) {
      error = 'Title is required';
      return;
    }

    saving = true;
    error = null;

    try {
      var startDateTime = allDay
        ? new Date(startDate + 'T00:00:00')
        : new Date(startDate + 'T' + (startTime || '09:00'));

      var endDateTime = null;
      if (endDate && (allDay || endTime)) {
        endDateTime = allDay
          ? new Date(endDate + 'T23:59:59')
          : new Date(endDate + 'T' + endTime);
      }

      var input = {
        title: title.trim(),
        description: description.trim(),
        eventType,
        startTime: startDateTime.toISOString(),
        endTime: endDateTime ? endDateTime.toISOString() : null,
        allDay,
        color
      };

      if ($selectedEvent?.id) {
        await updateEvent($selectedEvent.id, input);
      } else {
        await createEvent(input);
      }

      closeEventModal();
    } catch (err) {
      error = err.message || 'Failed to save event';
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!$selectedEvent?.id) return;
    if (!confirm('Delete this event?')) return;

    deleting = true;
    error = null;

    try {
      await deleteEvent($selectedEvent.id);
      closeEventModal();
    } catch (err) {
      error = err.message || 'Failed to delete event';
    } finally {
      deleting = false;
    }
  }

  function close() {
    closeEventModal();
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    }
  }

  var isEditMode = false;
  $: isEditMode = !!$selectedEvent?.id;
</script>

<svelte:window on:keydown={handleKeydown} />

{#if $showEventModal}
  <div class="overlay" on:click={close} role="button" tabindex="-1" on:keydown={handleKeydown}>
    <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
      <div class="modal-header">
        <h2>{isEditMode ? 'Edit Event' : 'New Event'}</h2>
        <button class="close-btn" on:click={close}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <div class="modal-content">
        {#if error}
          <div class="error-message">{error}</div>
        {/if}

        <div class="form-group">
          <label for="title">Title</label>
          <input
            type="text"
            id="title"
            bind:value={title}
            placeholder="Event title"
            autofocus
          />
        </div>

        <div class="form-group">
          <label for="eventType">Type</label>
          <select id="eventType" bind:value={eventType} on:change={handleEventTypeChange}>
            {#each Object.entries(eventTypeLabels) as [value, label]}
              <option {value}>{label}</option>
            {/each}
          </select>
        </div>

        <div class="form-row">
          <div class="form-group flex-1">
            <label for="startDate">Start Date</label>
            <input type="date" id="startDate" bind:value={startDate} />
          </div>
          {#if !allDay}
            <div class="form-group">
              <label for="startTime">Start Time</label>
              <input type="time" id="startTime" bind:value={startTime} />
            </div>
          {/if}
        </div>

        <div class="form-row">
          <div class="form-group flex-1">
            <label for="endDate">End Date</label>
            <input type="date" id="endDate" bind:value={endDate} />
          </div>
          {#if !allDay}
            <div class="form-group">
              <label for="endTime">End Time</label>
              <input type="time" id="endTime" bind:value={endTime} />
            </div>
          {/if}
        </div>

        <div class="form-group checkbox-group">
          <label>
            <input type="checkbox" bind:checked={allDay} />
            <span>All day event</span>
          </label>
        </div>

        <div class="form-group">
          <label for="color">Color</label>
          <div class="color-picker">
            <input type="color" id="color" bind:value={color} />
            <span class="color-preview" style="background: {color}"></span>
            <span class="color-value">{color}</span>
          </div>
        </div>

        <div class="form-group">
          <label for="description">Description</label>
          <textarea
            id="description"
            bind:value={description}
            placeholder="Optional description..."
            rows="3"
          ></textarea>
        </div>
      </div>

      <div class="modal-footer">
        {#if isEditMode}
          <button class="btn btn-danger" on:click={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting...' : 'Delete'}
          </button>
        {/if}
        <div class="spacer"></div>
        <button class="btn btn-secondary" on:click={close}>Cancel</button>
        <button class="btn btn-primary" on:click={handleSave} disabled={saving}>
          {saving ? 'Saving...' : (isEditMode ? 'Update' : 'Create')}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    width: 90%;
    max-width: 480px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .modal-header h2 {
    margin: 0;
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .modal-content {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-lg);
    display: flex;
    flex-direction: column;
    gap: var(--space-md);
  }

  .error-message {
    padding: var(--space-sm) var(--space-md);
    background: rgba(255, 87, 87, 0.1);
    border: 1px solid var(--accent-error);
    border-radius: var(--radius-sm);
    color: var(--accent-error);
    font-size: var(--font-sm);
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .form-group label {
    font-size: var(--font-sm);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .form-row {
    display: flex;
    gap: var(--space-md);
  }

  .flex-1 {
    flex: 1;
  }

  input[type="text"],
  input[type="date"],
  input[type="time"],
  select,
  textarea {
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: var(--font-sm);
    font-family: inherit;
    transition: border-color var(--transition-fast);
  }

  input:focus,
  select:focus,
  textarea:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  textarea {
    resize: vertical;
    min-height: 60px;
  }

  .checkbox-group label {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    cursor: pointer;
    flex-direction: row;
  }

  .checkbox-group input[type="checkbox"] {
    width: 16px;
    height: 16px;
    accent-color: var(--accent-primary);
  }

  .checkbox-group span {
    font-size: var(--font-sm);
    color: var(--text-primary);
  }

  .color-picker {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .color-picker input[type="color"] {
    width: 40px;
    height: 32px;
    padding: 2px;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    cursor: pointer;
  }

  .color-preview {
    width: 24px;
    height: 24px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--border-color);
  }

  .color-value {
    font-size: var(--font-sm);
    color: var(--text-muted);
    font-family: monospace;
  }

  .modal-footer {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-md) var(--space-lg);
    border-top: 1px solid var(--border-color);
  }

  .spacer {
    flex: 1;
  }

  .btn {
    padding: var(--space-sm) var(--space-lg);
    border: none;
    border-radius: var(--radius-sm);
    font-size: var(--font-sm);
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--accent-primary);
    color: var(--bg-primary);
  }

  .btn-primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-secondary {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--bg-hover);
  }

  .btn-danger {
    background: transparent;
    color: var(--accent-error);
    border: 1px solid var(--accent-error);
  }

  .btn-danger:hover:not(:disabled) {
    background: rgba(255, 87, 87, 0.1);
  }
</style>
