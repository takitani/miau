import { writable, get } from 'svelte/store';
import { selectNext, selectPrev, archiveEmail, deleteEmail, toggleStar, markAsRead, selectedEmailId, selectedEmail, emails, selectedIndex, loadEmails, refreshEmails, currentFolder, toggleThreading, restoreLastRemovedEmail } from './emails.js';
import { toggleDebug, info, warn, error as logError, debug as logDebug } from './debug.js';
import { toggleSelectionMode, selectAll, someSelected, exitSelectionMode, toggleSelection } from './selection.js';

// UI State
export const showSearch = writable(false);
export const showCompose = writable(false);
export const showHelp = writable(false);
export const showSettings = writable(false);
export const showAI = writable(false);
export const showAnalytics = writable(false);
export const aiWithContext = writable(false);

// Thread View State
export const showThreadView = writable(false);
export const threadEmailId = writable(null);

// Open thread view for an email
export function openThreadView(emailId) {
  threadEmailId.set(emailId);
  showThreadView.set(true);
}

// Close thread view
export function closeThreadView() {
  showThreadView.set(false);
  threadEmailId.set(null);
}

// Smart email selection - handles thread view transitions
// If thread view is open:
//   - If new email has thread (threadCount > 1): switch to that thread
//   - If new email has no thread: close thread view, show email
// If thread view is closed: just select the email
export function selectEmailSmart(id) {
  var $emails = get(emails);
  var email = $emails.find(e => e.id === id);
  var index = $emails.findIndex(e => e.id === id);

  if (index < 0 || !email) return;

  // Update selection
  selectedIndex.set(index);
  selectedEmailId.set(id);

  // Handle thread view
  var isThreadViewOpen = get(showThreadView);
  if (isThreadViewOpen) {
    if (email.threadCount > 1) {
      // Switch to new thread
      threadEmailId.set(id);
    } else {
      // Close thread view, show normal email
      closeThreadView();
    }
  }
}

// Active panel: 'folders' | 'emails' | 'viewer'
export const activePanel = writable('emails');

// Connection status
export const connected = writable(false);
export const lastSync = writable(null);
export const syncing = writable(false);

// Auto-refresh timer
export const autoRefreshInterval = 60; // seconds
export const autoRefreshStart = writable(Date.now());
export const autoRefreshEnabled = writable(false);
var autoRefreshTimer = null;

// New email notification
export const newEmailCount = writable(0);
export const newEmailShowUntil = writable(0);

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
  // Ignore if typing in an input field
  var isEditing = e.target.tagName === 'INPUT' ||
                  e.target.tagName === 'TEXTAREA' ||
                  e.target.isContentEditable;

  if (isEditing) {
    // Allow Escape to close modals
    if (e.key === 'Escape') {
      closeAllModals();
    }
    return;
  }

  // Also check if any modal is open - let the modal handle its own keys
  if (get(showCompose) || get(showAI) || get(showSearch)) {
    // Only handle Escape globally
    if (e.key === 'Escape') {
      closeAllModals();
    }
    return;
  }

  // Ctrl+Z / Ctrl+Y for undo/redo (works globally)
  if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
    e.preventDefault();
    performUndo();
    return;
  }
  if ((e.ctrlKey || e.metaKey) && (e.key === 'y' || (e.shiftKey && e.key === 'Z'))) {
    e.preventDefault();
    performRedo();
    return;
  }

  // Ctrl+A for select all (when not editing)
  if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
    e.preventDefault();
    selectAll();
    return;
  }

  // Global shortcuts (work everywhere)
  switch (e.key) {
    case 'Escape':
      // If in selection mode, exit it first
      if (get(someSelected)) {
        exitSelectionMode();
        return;
      }
      closeAllModals();
      return;

    case 'v':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        toggleSelectionMode();
        info('Modo de seleção: ' + (get(someSelected) ? 'ativado' : 'desativado'));
      }
      return;

    case ' ':
      // Space toggles current email selection
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        var emailId = get(selectedEmailId);
        var idx = get(selectedIndex);
        if (emailId) {
          toggleSelection(emailId, idx);
        }
      }
      return;

    case '/':
      e.preventDefault();
      showSearch.set(true);
      return;

    case '?':
      e.preventDefault();
      info('Help requested');
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

    case 'g':
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        toggleThreading();
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
        markAsRead(emailId, true);
        // Open thread view only if email has a thread (threadCount > 1)
        var emailForEnter = get(selectedEmail);
        if (emailForEnter && emailForEnter.threadCount > 1) {
          openThreadView(emailId);
        }
        // Otherwise just mark as read (email content shows in preview pane)
      }
      break;

    case 't':
      if (emailId && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        // Open thread view only if email has a thread (threadCount > 1)
        var emailForThread = get(selectedEmail);
        if (emailForThread && emailForThread.threadCount > 1) {
          openThreadView(emailId);
        } else {
          info('Email não faz parte de uma thread');
        }
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
  closeThreadView();
}

// Sync emails (current folder only)
export async function syncEmails() {
  syncing.set(true);
  info('Starting sync...');
  try {
    if (window.go?.desktop?.App) {
      var result = await window.go.desktop.App.SyncCurrentFolder();
      // Refresh emails without full reload (preserves selection, no flicker)
      var folder = get(currentFolder);
      await refreshEmails(folder);

      // Show notification (always, even if 0 new)
      var count = result ? result.newEmails : 0;
      newEmailCount.set(count);
      newEmailShowUntil.set(Date.now() + 3000); // 3 seconds
      if (count > 0) {
        info(`${count} novo(s) email(s)!`);
      } else {
        info('Nenhum email novo');
      }
    }
    lastSync.set(new Date());
    // Restart auto-refresh timer
    autoRefreshStart.set(Date.now());
    autoRefreshEnabled.set(true);
    startAutoRefreshTimer();
  } catch (err) {
    logError('Failed to sync', err);
  } finally {
    syncing.set(false);
  }
}

// Sync essential folders (INBOX, Sent, Trash)
export async function syncEssentialFolders() {
  syncing.set(true);
  info('Syncing essential folders (INBOX, Sent, Trash)...');
  try {
    if (window.go?.desktop?.App) {
      var results = await window.go.desktop.App.SyncEssentialFolders();
      // Refresh emails without full reload (preserves selection, no flicker)
      var folder = get(currentFolder);
      await refreshEmails(folder);

      // Sum all new emails
      var totalNew = 0;
      if (results) {
        for (var r of results) {
          totalNew += r.newEmails || 0;
        }
      }
      newEmailCount.set(totalNew);
      newEmailShowUntil.set(Date.now() + 3000);
      if (totalNew > 0) {
        info(`${totalNew} novo(s) email(s)!`);
      } else {
        info('Nenhum email novo');
      }
    }
    lastSync.set(new Date());
    autoRefreshStart.set(Date.now());
    autoRefreshEnabled.set(true);
    startAutoRefreshTimer();
  } catch (err) {
    logError('Failed to sync essential folders', err);
  } finally {
    syncing.set(false);
  }
}

// Start auto-refresh timer
function startAutoRefreshTimer() {
  if (autoRefreshTimer) clearInterval(autoRefreshTimer);

  autoRefreshTimer = setInterval(() => {
    if (!get(autoRefreshEnabled) || get(syncing)) return;

    var elapsed = (Date.now() - get(autoRefreshStart)) / 1000;
    // Add 1 second buffer to let progress bar complete visually
    if (elapsed >= autoRefreshInterval + 1) {
      info('Auto-refresh triggered');
      syncEmails();
    }
  }, 500);
}

// Stop auto-refresh timer
export function stopAutoRefreshTimer() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer);
    autoRefreshTimer = null;
  }
  autoRefreshEnabled.set(false);
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

    window.runtime.EventsOn('sync:completed', async (folder, newCount) => {
      syncing.set(false);
      lastSync.set(new Date());
      // Reload emails if sync brought new emails
      if (newCount > 0) {
        var current = get(currentFolder);
        if (folder === current)
          await loadEmails(current);
      }
    });

    window.runtime.EventsOn('sync:error', (error) => {
      syncing.set(false);
      console.error('Sync error:', error);
    });
  }
}

// Undo/Redo state
export const undoMessage = writable(null);

// Perform undo operation
async function performUndo() {
  try {
    if (window.go?.desktop?.App) {
      var result = await window.go.desktop.App.Undo();
      if (result.success) {
        info(result.description);
        // Restore email locally (no reload needed)
        restoreLastRemovedEmail();
      } else {
        warn(result.description);
      }
      showUndoMessage(result.description, result.success);
    }
  } catch (err) {
    logError('Undo failed', err);
  }
}

// Perform redo operation
async function performRedo() {
  try {
    if (window.go?.desktop?.App) {
      var result = await window.go.desktop.App.Redo();
      if (result.success) {
        info(result.description);
        // Reload emails to reflect the change
        var folder = get(currentFolder);
        if (folder) {
          await loadEmails(folder);
        }
      } else {
        warn(result.description);
      }
      showUndoMessage(result.description, result.success);
    }
  } catch (err) {
    logError('Redo failed', err);
  }
}

// Show temporary undo/redo message
function showUndoMessage(message, success) {
  undoMessage.set({ message, success });
  setTimeout(() => {
    undoMessage.set(null);
  }, 3000);
}
