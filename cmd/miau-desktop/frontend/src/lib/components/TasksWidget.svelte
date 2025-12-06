<script>
  import { onMount } from 'svelte';
  import { tasks, tasksLoading, taskCounts, loadPendingTasks, createTask, toggleTask, deleteTask, priorityColors } from '../stores/tasks.js';

  var newTaskTitle = '';
  var showAddInput = false;

  onMount(() => {
    loadPendingTasks();
  });

  async function handleAddTask(e) {
    e.preventDefault();
    if (!newTaskTitle.trim())
      return;

    try {
      await createTask(newTaskTitle.trim());
      newTaskTitle = '';
      showAddInput = false;
    } catch (err) {
      console.error('Failed to create task:', err);
    }
  }

  async function handleToggle(id) {
    try {
      await toggleTask(id);
    } catch (err) {
      console.error('Failed to toggle task:', err);
    }
  }

  async function handleDelete(id) {
    try {
      await deleteTask(id);
    } catch (err) {
      console.error('Failed to delete task:', err);
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      showAddInput = false;
      newTaskTitle = '';
    }
  }

  function getPriorityIndicator(priority) {
    if (priority === 2) return '!!';
    if (priority === 1) return '!';
    return '';
  }
</script>

<div class="tasks-widget">
  <!-- Header with count -->
  <div class="widget-header">
    <span class="task-count">
      {$taskCounts.pending} pending
    </span>
    <button class="add-btn" on:click={() => showAddInput = !showAddInput} title="Add task">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 5v14M5 12h14"/>
      </svg>
    </button>
  </div>

  <!-- Add task input -->
  {#if showAddInput}
    <form class="add-task-form" on:submit={handleAddTask}>
      <input
        type="text"
        bind:value={newTaskTitle}
        placeholder="New task..."
        class="add-task-input"
        on:keydown={handleKeydown}
        autofocus
      />
      <div class="add-task-actions">
        <button type="submit" class="action-btn save" disabled={!newTaskTitle.trim()}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 6L9 17l-5-5"/>
          </svg>
        </button>
        <button type="button" class="action-btn cancel" on:click={() => { showAddInput = false; newTaskTitle = ''; }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>
    </form>
  {/if}

  <!-- Task list -->
  <div class="task-list">
    {#if $tasksLoading}
      <div class="loading">Loading...</div>
    {:else if $tasks.length === 0}
      <div class="empty">
        <span class="empty-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 11l3 3L22 4"/>
            <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
          </svg>
        </span>
        <span>No tasks</span>
      </div>
    {:else}
      {#each $tasks.slice(0, 5) as task (task.id)}
        <div class="task-item" class:completed={task.isCompleted}>
          <button class="checkbox" on:click={() => handleToggle(task.id)}>
            {#if task.isCompleted}
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <path d="M20 6L9 17l-5-5"/>
              </svg>
            {/if}
          </button>
          <span class="task-title" style="--priority-color: {priorityColors[task.priority]}">
            {#if task.priority > 0}
              <span class="priority-indicator" style="color: {priorityColors[task.priority]}">
                {getPriorityIndicator(task.priority)}
              </span>
            {/if}
            {task.title}
          </span>
          <button class="delete-btn" on:click={() => handleDelete(task.id)} title="Delete">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6L6 18M6 6l12 12"/>
            </svg>
          </button>
        </div>
      {/each}
      {#if $tasks.length > 5}
        <div class="more-tasks">
          +{$tasks.length - 5} more
        </div>
      {/if}
    {/if}
  </div>
</div>

<style>
  .tasks-widget {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .widget-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--space-xs);
  }

  .task-count {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .add-btn {
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

  .add-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-primary);
  }

  .add-task-form {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs);
    background: var(--bg-tertiary);
    border-radius: var(--radius-sm);
  }

  .add-task-input {
    flex: 1;
    padding: var(--space-xs) var(--space-sm);
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-size: var(--font-sm);
    outline: none;
  }

  .add-task-input::placeholder {
    color: var(--text-muted);
  }

  .add-task-actions {
    display: flex;
    gap: 2px;
  }

  .action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .action-btn.save {
    color: var(--accent-success);
  }

  .action-btn.save:disabled {
    color: var(--text-muted);
    cursor: not-allowed;
  }

  .action-btn.cancel {
    color: var(--text-secondary);
  }

  .action-btn:hover:not(:disabled) {
    background: var(--bg-hover);
  }

  .task-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .task-item {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs);
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
  }

  .task-item:hover {
    background: var(--bg-hover);
  }

  .task-item:hover .delete-btn {
    opacity: 1;
  }

  .checkbox {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border: 1.5px solid var(--text-muted);
    border-radius: 3px;
    background: transparent;
    cursor: pointer;
    color: var(--accent-success);
    flex-shrink: 0;
    transition: all var(--transition-fast);
  }

  .task-item.completed .checkbox {
    background: var(--accent-success);
    border-color: var(--accent-success);
    color: var(--bg-primary);
  }

  .task-title {
    flex: 1;
    font-size: var(--font-sm);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .task-item.completed .task-title {
    color: var(--text-muted);
    text-decoration: line-through;
  }

  .priority-indicator {
    font-weight: 700;
    margin-right: 4px;
  }

  .delete-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: none;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: var(--radius-sm);
    opacity: 0;
    transition: all var(--transition-fast);
  }

  .delete-btn:hover {
    background: var(--bg-hover);
    color: var(--accent-error);
  }

  .loading, .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-md);
    color: var(--text-muted);
    font-size: var(--font-sm);
    text-align: center;
  }

  .empty-icon {
    opacity: 0.5;
  }

  .more-tasks {
    font-size: var(--font-xs);
    color: var(--text-muted);
    text-align: center;
    padding: var(--space-xs);
  }
</style>
