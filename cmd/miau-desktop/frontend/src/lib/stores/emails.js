import { writable, derived, get } from 'svelte/store';
import { info, error as logError, debug as logDebug } from './debug.js';

// Email list
export const emails = writable([]);

// Selected email ID
export const selectedEmailId = writable(null);

// Selected email index for keyboard navigation
export const selectedIndex = writable(0);

// Current folder
export const currentFolder = writable('INBOX');

// Loading state
export const loading = writable(false);

// Threading enabled (groups emails by thread, showing only latest with count)
export const threadingEnabled = writable(true);

// Stack of recently removed emails (for undo without reload)
let recentlyRemovedEmails = [];

// Derived: get selected email object
export const selectedEmail = derived(
  [emails, selectedEmailId],
  ([$emails, $id]) => $emails.find(e => e.id === $id) || null
);

// Load emails from backend
export async function loadEmails(folder, limit = 50) {
  loading.set(true);
  const threaded = get(threadingEnabled);
  logDebug(`loadEmails called: folder=${folder}, limit=${limit}, threaded=${threaded}`);
  try {
    // Check if Wails bindings are available
    if (typeof window !== 'undefined' && window.go && window.go.desktop && window.go.desktop.App) {
      let result;
      if (threaded) {
        logDebug('Calling Go backend GetEmailsThreaded...');
        result = await window.go.desktop.App.GetEmailsThreaded(folder, limit);
      } else {
        logDebug('Calling Go backend GetEmails...');
        result = await window.go.desktop.App.GetEmails(folder, limit);
      }
      logDebug(`GetEmails returned ${result ? result.length : 0} emails`);
      emails.set(result || []);

      // Select first email if available
      if (result && result.length > 0) {
        selectedEmailId.set(result[0].id);
        selectedIndex.set(0);
        info(`Loaded ${result.length} ${threaded ? 'threads' : 'emails'} from ${folder}`);
      } else {
        info(`No emails found in ${folder}`);
      }
    } else {
      // Mock data for development
      logDebug('Wails bindings not available, using mock data');
      emails.set(getMockEmails());
      selectedEmailId.set(1);
      selectedIndex.set(0);
    }
  } catch (err) {
    logError(`Failed to load emails from ${folder}`, err);
    emails.set([]);
  } finally {
    loading.set(false);
  }
}

// Toggle threading mode and reload emails
export async function toggleThreading() {
  const current = get(threadingEnabled);
  threadingEnabled.set(!current);
  const folder = get(currentFolder);
  await loadEmails(folder);
}

// Refresh emails without full reload (merge new emails, preserve selection)
export async function refreshEmails(folder, limit = 50) {
  const threaded = get(threadingEnabled);
  logDebug(`refreshEmails called: folder=${folder}, limit=${limit}, threaded=${threaded}`);

  try {
    if (typeof window !== 'undefined' && window.go && window.go.desktop && window.go.desktop.App) {
      let result;
      if (threaded) {
        result = await window.go.desktop.App.GetEmailsThreaded(folder, limit);
      } else {
        result = await window.go.desktop.App.GetEmails(folder, limit);
      }

      const newEmails = result || [];
      const currentEmails = get(emails);
      const currentSelectedId = get(selectedEmailId);

      // Check if list actually changed
      const hasChanges = newEmails.length !== currentEmails.length ||
        newEmails.some((e, i) => !currentEmails[i] || e.id !== currentEmails[i].id);

      if (!hasChanges) {
        logDebug('No changes detected, skipping update');
        return 0;
      }

      // Count new emails (IDs that weren't in the previous list)
      const oldIds = new Set(currentEmails.map(e => e.id));
      const newCount = newEmails.filter(e => !oldIds.has(e.id)).length;

      // Update list
      emails.set(newEmails);

      // Preserve selection if email still exists
      if (currentSelectedId) {
        const stillExists = newEmails.find(e => e.id === currentSelectedId);
        if (stillExists) {
          const newIndex = newEmails.findIndex(e => e.id === currentSelectedId);
          selectedIndex.set(newIndex);
          // Keep selectedEmailId as is
        } else if (newEmails.length > 0) {
          // Selected email was removed, select first
          selectedEmailId.set(newEmails[0].id);
          selectedIndex.set(0);
        }
      } else if (newEmails.length > 0) {
        // No selection, select first
        selectedEmailId.set(newEmails[0].id);
        selectedIndex.set(0);
      }

      logDebug(`Refreshed: ${newCount} new emails, ${newEmails.length} total`);
      return newCount;
    }
  } catch (err) {
    logError(`Failed to refresh emails from ${folder}`, err);
  }
  return 0;
}

// Navigate to next email
export function selectNext() {
  const $emails = get(emails);
  const $index = get(selectedIndex);

  if ($index < $emails.length - 1) {
    const newIndex = $index + 1;
    selectedIndex.set(newIndex);
    selectedEmailId.set($emails[newIndex].id);
  }
}

// Navigate to previous email
export function selectPrev() {
  const $emails = get(emails);
  const $index = get(selectedIndex);

  if ($index > 0) {
    const newIndex = $index - 1;
    selectedIndex.set(newIndex);
    selectedEmailId.set($emails[newIndex].id);
  }
}

// Select email by ID
// If the email is not in the current list (e.g., from search results for older emails),
// fetch it from the backend and prepend to the list
export async function selectEmail(id) {
  const $emails = get(emails);
  const index = $emails.findIndex(e => e.id === id);

  if (index >= 0) {
    // Email is in the list, just select it
    selectedIndex.set(index);
    selectedEmailId.set(id);
  } else {
    // Email not in current list - fetch it from backend and prepend
    try {
      if (window.go?.desktop?.App) {
        logDebug(`Email ${id} not in list, fetching from backend...`);
        const email = await window.go.desktop.App.GetEmailByID(id);
        if (email) {
          // Prepend to list so it becomes visible
          emails.update(list => [email, ...list]);
          selectedIndex.set(0);
          selectedEmailId.set(id);
          info(`Loaded email from search: ${email.subject}`);
        } else {
          logError(`Email ${id} not found in backend`);
        }
      }
    } catch (err) {
      logError(`Failed to fetch email ${id}`, err);
    }
  }
}

// Mark email as read
export async function markAsRead(id, read = true) {
  try {
    if (window.go?.desktop?.App) {
      await window.go.desktop.App.MarkAsRead(id, read);
    }
    // Update local state
    emails.update(list =>
      list.map(e => e.id === id ? { ...e, isRead: read } : e)
    );
  } catch (err) {
    console.error('Failed to mark as read:', err);
  }
}

// Archive email
export async function archiveEmail(id) {
  try {
    // Save email before removing (for undo)
    const $emails = get(emails);
    const email = $emails.find(e => e.id === id);
    const index = $emails.findIndex(e => e.id === id);
    if (email) {
      recentlyRemovedEmails.push({ email, index, action: 'archive' });
      // Keep max 50 items
      if (recentlyRemovedEmails.length > 50) recentlyRemovedEmails.shift();
    }

    if (window.go?.desktop?.App) {
      await window.go.desktop.App.Archive(id);
    }
    // Remove from list
    emails.update(list => list.filter(e => e.id !== id));

    // Select next email
    selectNext();
  } catch (err) {
    console.error('Failed to archive:', err);
  }
}

// Delete email
export async function deleteEmail(id) {
  try {
    // Save email before removing (for undo)
    const $emails = get(emails);
    const email = $emails.find(e => e.id === id);
    const index = $emails.findIndex(e => e.id === id);
    if (email) {
      recentlyRemovedEmails.push({ email, index, action: 'delete' });
      // Keep max 50 items
      if (recentlyRemovedEmails.length > 50) recentlyRemovedEmails.shift();
    }

    if (window.go?.desktop?.App) {
      await window.go.desktop.App.Delete(id);
    }
    // Remove from list
    emails.update(list => list.filter(e => e.id !== id));

    // Select next email
    selectNext();
  } catch (err) {
    console.error('Failed to delete:', err);
  }
}

// Restore last removed email (for undo without reload)
export function restoreLastRemovedEmail() {
  if (recentlyRemovedEmails.length === 0) return null;

  const removed = recentlyRemovedEmails.pop();
  if (!removed) return null;

  // Insert email back at original position
  emails.update(list => {
    const newList = [...list];
    const insertIndex = Math.min(removed.index, newList.length);
    newList.splice(insertIndex, 0, removed.email);
    return newList;
  });

  // Select the restored email
  selectedEmailId.set(removed.email.id);
  selectedIndex.set(removed.index);

  return removed.email;
}

// Toggle star
export async function toggleStar(id) {
  const $emails = get(emails);
  const email = $emails.find(e => e.id === id);
  if (!email) return;

  const newStarred = !email.isStarred;

  try {
    if (window.go?.desktop?.App) {
      await window.go.desktop.App.MarkAsStarred(id, newStarred);
    }
    // Update local state
    emails.update(list =>
      list.map(e => e.id === id ? { ...e, isStarred: newStarred } : e)
    );
  } catch (err) {
    console.error('Failed to toggle star:', err);
  }
}

// Mock data for development without backend
function getMockEmails() {
  return [
    {
      id: 1,
      uid: 1001,
      subject: 'Welcome to miau!',
      fromName: 'miau Team',
      fromEmail: 'team@miau.app',
      date: new Date().toISOString(),
      isRead: false,
      isStarred: true,
      hasAttachments: false,
      snippet: 'Thank you for trying miau, your new email client...'
    },
    {
      id: 2,
      uid: 1002,
      subject: 'Re: Project Proposal',
      fromName: 'Maria Silva',
      fromEmail: 'maria@example.com',
      date: new Date(Date.now() - 3600000).toISOString(),
      isRead: false,
      isStarred: false,
      hasAttachments: true,
      snippet: 'I reviewed the proposal and have some feedback...'
    },
    {
      id: 3,
      uid: 1003,
      subject: 'Meeting Tomorrow',
      fromName: 'John Santos',
      fromEmail: 'john@example.com',
      date: new Date(Date.now() - 7200000).toISOString(),
      isRead: true,
      isStarred: false,
      hasAttachments: false,
      snippet: 'Can we meet at 2pm to discuss the project?'
    },
    {
      id: 4,
      uid: 1004,
      subject: 'Invoice #12345',
      fromName: 'Finance',
      fromEmail: 'finance@company.com',
      date: new Date(Date.now() - 86400000).toISOString(),
      isRead: true,
      isStarred: false,
      hasAttachments: true,
      snippet: 'Please find attached your invoice for December...'
    }
  ];
}
