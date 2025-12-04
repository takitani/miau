import { writable, get } from 'svelte/store';

// Debug mode enabled
export const debugEnabled = writable(false);

// Log entries
export const logs = writable([]);

// Max log entries to keep
const MAX_LOGS = 500;

// Log levels
export const LogLevel = {
  DEBUG: 'debug',
  INFO: 'info',
  WARN: 'warn',
  ERROR: 'error',
  EVENT: 'event'
};

// Add a log entry
export function log(level, message, data = null) {
  const entry = {
    id: Date.now() + Math.random(),
    timestamp: new Date().toISOString().substr(11, 12),
    level,
    message,
    data
  };

  logs.update(list => {
    const newList = [entry, ...list];
    if (newList.length > MAX_LOGS) {
      newList.pop();
    }
    return newList;
  });

  // Also log to console in dev
  const consoleMethod = level === 'error' ? 'error' : level === 'warn' ? 'warn' : 'log';
  console[consoleMethod](`[${entry.timestamp}] [${level.toUpperCase()}] ${message}`, data || '');
}

// Convenience methods
export const debug = (msg, data) => log(LogLevel.DEBUG, msg, data);
export const info = (msg, data) => log(LogLevel.INFO, msg, data);
export const warn = (msg, data) => log(LogLevel.WARN, msg, data);
export const error = (msg, data) => log(LogLevel.ERROR, msg, data);
export const event = (msg, data) => log(LogLevel.EVENT, msg, data);

// Toggle debug panel
export function toggleDebug() {
  debugEnabled.update(v => !v);
}

// Clear logs
export function clearLogs() {
  logs.set([]);
}

// Setup event listeners from Wails
export function setupDebugEvents() {
  if (typeof window === 'undefined') return;

  // Intercept Wails runtime events
  const runtime = window.runtime;
  if (!runtime) return;

  // Listen to all events from Go backend
  const events = [
    'email:new',
    'email:read',
    'sync:started',
    'sync:completed',
    'sync:error',
    'connection:connected',
    'connection:disconnected',
    'connection:error',
    'send:completed',
    'bounce:detected',
    'batch:created',
    'index:progress'
  ];

  events.forEach(eventName => {
    runtime.EventsOn(eventName, (...args) => {
      event(`Event: ${eventName}`, args.length === 1 ? args[0] : args);
    });
  });

  info('Debug events initialized');
}
