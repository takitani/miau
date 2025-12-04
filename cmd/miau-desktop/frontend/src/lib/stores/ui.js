import { writable, get } from 'svelte/store';
import { selectNext, selectPrev, archiveEmail, deleteEmail, toggleStar, markAsRead, selectedEmailId, selectedEmail } from './emails.js';
import { toggleDebug, info, warn, error as logError, debug as logDebug } from './debug.js';

// UI State
export const showSearch = writable(false);
export const showCompose = writable(false);
export const showHelp = writable(false);
export const showSettings = writable(false);
export const showAI = writable(false);
export const showAnalytics = writable(false);
export const aiWithContext = writable(false);

// Active panel: 'folders' | 'emails' | 'viewer'
export const activePanel = writable('emails');

// Connection status
export const connected = writable(false);
export const lastSync = writable(null);
export const syncing = writable(false);

// AI Providers - CLI based
export const aiProviders = writable([
  { id: 'claude', name: 'Claude', icon: '🤖', cmd: 'claude' },
  { id: 'gemini', name: 'Gemini', icon: '✨', cmd: 'gemini' },
  { id: 'ollama', name: 'Ollama', icon: '🦙', cmd: 'ollama' },
  { id: 'openai', name: 'OpenAI', icon: '🧠', cmd: 'openai' },
]);
export const aiProvider = writable('claude');

// Setup keyboard shortcuts
export function setupKeyboardShortcuts() {
  document.addEventListener('keydown', handleKeydown);
}

// Handle keyboard events
function handleKeydown(e) {
  // Ignore if typing in an input
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
    // Allow Escape to close modals
    if (e.key === 'Escape') {
      closeAllModals();
    }
    return;
  }

  // Global shortcuts (work everywhere)
  switch (e.key) {
    case 'Escape':
      closeAllModals();
      return;

    case '/':
      e.preventDefault();
      showSearch.set(true);
      return;

    case '?':
      e.preventDefault();
      showHelp.set(true);
      return;

    case 'c':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        showCompose.set(true);
      }
      return;

    case 'S':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        showSettings.set(true);
      }
      return;

    case 'D':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        toggleDebug();
        logDebug('Debug panel toggled');
      }
      return;

    case 'T':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        switchToTerminal();
      }
      return;

    case 'a':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        aiWithContext.set(false);
        showAI.set(true);
      }
      return;

    case 'A':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        aiWithContext.set(true);
        showAI.set(true);
      }
      return;

    case 'p':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        showAnalytics.update(v => !v);
      }
      return;
  }

  // Panel-specific shortcuts
  const panel = get(activePanel);

  if (panel === 'emails' || panel === 'viewer') {
    handleEmailShortcuts(e);
  }

  // Panel navigation
  if (e.key === 'Tab') {
    e.preventDefault();
    cyclePanels(e.shiftKey ? -1 : 1);
  }
}

// Handle email-related shortcuts
function handleEmailShortcuts(e) {
  const emailId = get(selectedEmailId);

  switch (e.key) {
    // Navigation
    case 'j':
    case 'ArrowDown':
      e.preventDefault();
      selectNext();
      break;

    case 'k':
    case 'ArrowUp':
      e.preventDefault();
      selectPrev();
      break;

    // Actions
    case 'e':
      if (emailId && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        archiveEmail(emailId);
      }
      break;

    case 'x':
    case '#':
      if (emailId && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        deleteEmail(emailId);
      }
      break;

    case 's':
      if (emailId && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        toggleStar(emailId);
      }
      break;

    case 'u':
      if (emailId && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        markAsRead(emailId, false);
      }
      break;

    case 'Enter':
      if (emailId) {
        e.preventDefault();
        // Switch to viewer panel and mark as read
        activePanel.set('viewer');
        markAsRead(emailId, true);
      }
      break;

    // Reply (when email selected) or Refresh (no selection)
    case 'r':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        var email = get(selectedEmail);
        if (email) {
          // Reply to selected email
          window.composeContext = { mode: 'reply', replyTo: email };
          showCompose.set(true);
        } else {
          // No email selected, sync
          syncEmails();
        }
      }
      break;

    // Reply All
    case 'R':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        var email = get(selectedEmail);
        if (email) {
          window.composeContext = { mode: 'replyAll', replyTo: email };
          showCompose.set(true);
        }
      }
      break;

    // Forward
    case 'f':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        var email = get(selectedEmail);
        if (email) {
          window.composeContext = { mode: 'forward', forwardEmail: email };
          showCompose.set(true);
        }
      }
      break;

    // Go to shortcuts (g + key)
    case 'g':
      // TODO: implement go-to mode
      break;
  }
}

// Cycle through panels
function cyclePanels(direction) {
  const panels = ['folders', 'emails', 'viewer'];
  const current = get(activePanel);
  const currentIndex = panels.indexOf(current);
  const newIndex = (currentIndex + direction + panels.length) % panels.length;
  activePanel.set(panels[newIndex]);
}

// Close all modals
function closeAllModals() {
  showSearch.set(false);
  showCompose.set(false);
  showHelp.set(false);
  showSettings.set(false);
  showAI.set(false);
  showAnalytics.set(false);
}

// Sync emails
export async function syncEmails() {
  syncing.set(true);
  info('Starting sync...');
  try {
    if (window.go?.desktop?.App) {
      await window.go.desktop.App.SyncCurrentFolder();
    }
    lastSync.set(new Date());
    info('Sync completed');
  } catch (err) {
    logError('Failed to sync', err);
  } finally {
    syncing.set(false);
  }
}

// Connect to server
export async function connect() {
  try {
    if (window.go?.desktop?.App) {
      await window.go.desktop.App.Connect();
      connected.set(true);
    }
  } catch (err) {
    console.error('Failed to connect:', err);
    connected.set(false);
  }
}

// Disconnect from server
export async function disconnect() {
  try {
    if (window.go?.desktop?.App) {
      await window.go.desktop.App.Disconnect();
    }
    connected.set(false);
  } catch (err) {
    console.error('Failed to disconnect:', err);
  }
}

// Switch to terminal mode
export async function switchToTerminal() {
  info('Switching to terminal mode...');
  try {
    if (window.go?.desktop?.App) {
      await window.go.desktop.App.SwitchToTerminal();
      info('Terminal launched');
    }
  } catch (err) {
    logError('Failed to switch to terminal', err);
  }
}

// Setup Wails event listeners
export function setupWailsEvents() {
  if (typeof window !== 'undefined' && window.runtime) {
    window.runtime.EventsOn('connection:connected', () => {
      connected.set(true);
    });

    window.runtime.EventsOn('connection:disconnected', () => {
      connected.set(false);
    });

    window.runtime.EventsOn('sync:started', (folder) => {
      syncing.set(true);
    });

    window.runtime.EventsOn('sync:completed', (folder, newCount) => {
      syncing.set(false);
      lastSync.set(new Date());
    });

    window.runtime.EventsOn('sync:error', (error) => {
      syncing.set(false);
      console.error('Sync error:', error);
    });
  }
}
