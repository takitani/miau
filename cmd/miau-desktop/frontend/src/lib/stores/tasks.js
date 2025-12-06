import { writable, get } from 'svelte/store';
import { GetTasks, GetPendingTasks, CreateTask, UpdateTask, ToggleTaskComplete, DeleteTask, GetTaskCounts } from '../wailsjs/wailsjs/go/desktop/App.js';

// Task list store
export var tasks = writable([]);
export var tasksLoading = writable(false);
export var taskCounts = writable({ pending: 0, completed: 0, total: 0 });

// Load all tasks
export async function loadTasks() {
  tasksLoading.set(true);
  try {
    var result = await GetTasks();
    tasks.set(result || []);
    await loadTaskCounts();
  } catch (err) {
    console.error('Failed to load tasks:', err);
    tasks.set([]);
  } finally {
    tasksLoading.set(false);
  }
}

// Load only pending tasks
export async function loadPendingTasks() {
  tasksLoading.set(true);
  try {
    var result = await GetPendingTasks();
    tasks.set(result || []);
    await loadTaskCounts();
  } catch (err) {
    console.error('Failed to load pending tasks:', err);
    tasks.set([]);
  } finally {
    tasksLoading.set(false);
  }
}

// Load task counts
export async function loadTaskCounts() {
  try {
    var counts = await GetTaskCounts();
    if (counts)
      taskCounts.set(counts);
  } catch (err) {
    console.error('Failed to load task counts:', err);
  }
}

// Create a new task
export async function createTask(title, description = '', priority = 0, dueDate = null, emailId = null, source = 'manual') {
  try {
    var input = {
      title,
      description,
      isCompleted: false,
      priority,
      dueDate,
      emailId,
      source
    };
    var newTask = await CreateTask(input);
    if (newTask) {
      tasks.update(list => [newTask, ...list]);
      await loadTaskCounts();
    }
    return newTask;
  } catch (err) {
    console.error('Failed to create task:', err);
    throw err;
  }
}

// Update an existing task
export async function updateTask(id, updates) {
  try {
    var currentTasks = get(tasks);
    var existing = currentTasks.find(t => t.id === id);
    if (!existing)
      throw new Error('Task not found');

    var input = {
      id,
      title: updates.title ?? existing.title,
      description: updates.description ?? existing.description,
      isCompleted: updates.isCompleted ?? existing.isCompleted,
      priority: updates.priority ?? existing.priority,
      dueDate: updates.dueDate ?? existing.dueDate,
      emailId: updates.emailId ?? existing.emailId,
      source: updates.source ?? existing.source
    };

    var updated = await UpdateTask(input);
    if (updated) {
      tasks.update(list => list.map(t => t.id === id ? updated : t));
      await loadTaskCounts();
    }
    return updated;
  } catch (err) {
    console.error('Failed to update task:', err);
    throw err;
  }
}

// Toggle task completion
export async function toggleTask(id) {
  try {
    var newStatus = await ToggleTaskComplete(id);
    tasks.update(list => list.map(t => {
      if (t.id === id)
        return { ...t, isCompleted: newStatus };
      return t;
    }));
    await loadTaskCounts();
    return newStatus;
  } catch (err) {
    console.error('Failed to toggle task:', err);
    throw err;
  }
}

// Delete a task
export async function deleteTask(id) {
  try {
    await DeleteTask(id);
    tasks.update(list => list.filter(t => t.id !== id));
    await loadTaskCounts();
  } catch (err) {
    console.error('Failed to delete task:', err);
    throw err;
  }
}

// Get task by ID
export function getTask(id) {
  return get(tasks).find(t => t.id === id);
}

// Priority labels
export var priorityLabels = {
  0: 'Normal',
  1: 'High',
  2: 'Urgent'
};

export var priorityColors = {
  0: 'var(--text-secondary)',
  1: 'var(--accent-warning)',
  2: 'var(--accent-error)'
};
