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

// Derived: get selected email object
export const selectedEmail = derived(
  [emails, selectedEmailId],
  ([$emails, $id]) => $emails.find(e => e.id === $id) || null
);

// Load emails from backend
export async function loadEmails(folder, limit = 50) {
  loading.set(true);
  logDebug(`loadEmails called: folder=${folder}, limit=${limit}`);
  try {
    // Check if Wails bindings are available
    if (typeof window !== 'undefined' && window.go && window.go.desktop && window.go.desktop.App) {
      logDebug('Calling Go backend GetEmails...');
      const result = await window.go.desktop.App.GetEmails(folder, limit);
      logDebug(`GetEmails returned ${result ? result.length : 0} emails`);
      emails.set(result || []);

      // Select first email if available
      if (result && result.length > 0) {
        selectedEmailId.set(result[0].id);
        selectedIndex.set(0);
        info(`Loaded ${result.length} emails from ${folder}`);
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
export function selectEmail(id) {
  const $emails = get(emails);
  const index = $emails.findIndex(e => e.id === id);

  if (index >= 0) {
    selectedIndex.set(index);
    selectedEmailId.set(id);
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
