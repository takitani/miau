import { writable, derived, get } from 'svelte/store';
import { emails } from './emails.js';
import { info, warn } from './debug.js';

// Selection state
export const selectedIds = writable(new Set());
export const selectionMode = writable(false); // Visual mode active
export const lastSelectedIndex = writable(-1); // For shift+click range selection
export const dragSelecting = writable(false); // Drag selection in progress
export const dragStartIndex = writable(-1);

// Derived: selected count
export const selectedCount = derived(selectedIds, $ids => $ids.size);

// Derived: selected emails
export const selectedEmails = derived(
  [emails, selectedIds],
  ([$emails, $ids]) => $emails.filter(e => $ids.has(e.id))
);

// Derived: all selected?
export const allSelected = derived(
  [emails, selectedIds],
  ([$emails, $ids]) => $emails.length > 0 && $emails.every(e => $ids.has(e.id))
);

// Derived: some selected?
export const someSelected = derived(selectedIds, $ids => $ids.size > 0);

// Toggle selection mode
export function toggleSelectionMode() {
  selectionMode.update(v => {
    if (v) {
      // Exiting selection mode - clear selection
      clearSelection();
    }
    return !v;
  });
}

// Enter selection mode
export function enterSelectionMode() {
  selectionMode.set(true);
}

// Exit selection mode and clear
export function exitSelectionMode() {
  selectionMode.set(false);
  clearSelection();
}

// Toggle single email selection
export function toggleSelection(id, index = -1) {
  selectedIds.update(ids => {
    const newIds = new Set(ids);
    if (newIds.has(id)) {
      newIds.delete(id);
    } else {
      newIds.add(id);
    }
    return newIds;
  });

  if (index >= 0) {
    lastSelectedIndex.set(index);
  }

  // Auto-enter selection mode if selecting
  if (get(selectedIds).size > 0) {
    selectionMode.set(true);
  }
}

// Select single (replace selection)
export function selectSingle(id, index = -1) {
  selectedIds.set(new Set([id]));
  if (index >= 0) {
    lastSelectedIndex.set(index);
  }
  selectionMode.set(true);
}

// Range selection (shift+click)
export function selectRange(toIndex) {
  let fromIndex = get(lastSelectedIndex);
  if (fromIndex < 0) fromIndex = 0;

  const $emails = get(emails);
  const start = Math.min(fromIndex, toIndex);
  const end = Math.max(fromIndex, toIndex);

  selectedIds.update(ids => {
    const newIds = new Set(ids);
    for (let i = start; i <= end; i++) {
      if ($emails[i]) {
        newIds.add($emails[i].id);
      }
    }
    return newIds;
  });

  lastSelectedIndex.set(toIndex);
  selectionMode.set(true);
}

// Select all visible emails
export function selectAll() {
  const $emails = get(emails);
  selectedIds.set(new Set($emails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected all ${$emails.length} emails`);
}

// Clear selection
export function clearSelection() {
  selectedIds.set(new Set());
  lastSelectedIndex.set(-1);
}

// Invert selection
export function invertSelection() {
  const $emails = get(emails);
  const $selectedIds = get(selectedIds);

  const newIds = new Set();
  for (const e of $emails) {
    if (!$selectedIds.has(e.id)) {
      newIds.add(e.id);
    }
  }

  selectedIds.set(newIds);
  if (newIds.size > 0) {
    selectionMode.set(true);
  }
}

// Smart selection: by sender
export function selectBySender(senderEmail) {
  const $emails = get(emails);
  const matching = $emails.filter(e => e.fromEmail === senderEmail);

  selectedIds.update(ids => {
    const newIds = new Set(ids);
    for (const e of matching) {
      newIds.add(e.id);
    }
    return newIds;
  });

  selectionMode.set(true);
  info(`Selected ${matching.length} emails from ${senderEmail}`);
}

// Smart selection: unread
export function selectUnread() {
  const $emails = get(emails);
  const unread = $emails.filter(e => !e.isRead);

  selectedIds.set(new Set(unread.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${unread.length} unread emails`);
}

// Smart selection: read
export function selectRead() {
  const $emails = get(emails);
  const read = $emails.filter(e => e.isRead);

  selectedIds.set(new Set(read.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${read.length} read emails`);
}

// Smart selection: with attachments
export function selectWithAttachments() {
  const $emails = get(emails);
  const withAtt = $emails.filter(e => e.hasAttachments);

  selectedIds.set(new Set(withAtt.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${withAtt.length} emails with attachments`);
}

// Smart selection: starred
export function selectStarred() {
  const $emails = get(emails);
  const starred = $emails.filter(e => e.isStarred);

  selectedIds.set(new Set(starred.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${starred.length} starred emails`);
}

// Smart selection: today
export function selectToday() {
  const $emails = get(emails);
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const todayEmails = $emails.filter(e => {
    const emailDate = new Date(e.date);
    emailDate.setHours(0, 0, 0, 0);
    return emailDate.getTime() === today.getTime();
  });

  selectedIds.set(new Set(todayEmails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${todayEmails.length} emails from today`);
}

// Smart selection: this week
export function selectThisWeek() {
  const $emails = get(emails);
  const now = new Date();
  const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

  const weekEmails = $emails.filter(e => new Date(e.date) >= weekAgo);

  selectedIds.set(new Set(weekEmails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${weekEmails.length} emails from this week`);
}

// Smart selection: older than a week
export function selectOlderThanWeek() {
  const $emails = get(emails);
  const now = new Date();
  const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

  const oldEmails = $emails.filter(e => new Date(e.date) < weekAgo);

  selectedIds.set(new Set(oldEmails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${oldEmails.length} emails older than a week`);
}

// Drag selection helpers
export function startDragSelection(index) {
  dragSelecting.set(true);
  dragStartIndex.set(index);
  const $emails = get(emails);
  if ($emails[index]) {
    selectedIds.set(new Set([$emails[index].id]));
  }
  selectionMode.set(true);
}

export function updateDragSelection(currentIndex) {
  if (!get(dragSelecting)) return;

  const startIdx = get(dragStartIndex);
  const $emails = get(emails);
  const start = Math.min(startIdx, currentIndex);
  const end = Math.max(startIdx, currentIndex);

  const newIds = new Set();
  for (let i = start; i <= end; i++) {
    if ($emails[i]) {
      newIds.add($emails[i].id);
    }
  }

  selectedIds.set(newIds);
}

export function endDragSelection() {
  dragSelecting.set(false);
  dragStartIndex.set(-1);
  lastSelectedIndex.set(-1);
}

// Batch operations (call backend)
export async function batchArchive() {
  const ids = Array.from(get(selectedIds));
  if (ids.length === 0) return;

  try {
    if (window.go?.desktop?.App?.BatchArchive) {
      await window.go.desktop.App.BatchArchive(ids);
      info(`Archived ${ids.length} emails`);

      // Remove from local list
      emails.update(list => list.filter(e => !get(selectedIds).has(e.id)));
      clearSelection();
      exitSelectionMode();
    }
  } catch (err) {
    warn(`Failed to archive: ${err}`);
  }
}

export async function batchDelete() {
  const ids = Array.from(get(selectedIds));
  if (ids.length === 0) return;

  try {
    if (window.go?.desktop?.App?.BatchDelete) {
      await window.go.desktop.App.BatchDelete(ids);
      info(`Deleted ${ids.length} emails`);

      emails.update(list => list.filter(e => !get(selectedIds).has(e.id)));
      clearSelection();
      exitSelectionMode();
    }
  } catch (err) {
    warn(`Failed to delete: ${err}`);
  }
}

export async function batchMarkRead() {
  const ids = Array.from(get(selectedIds));
  if (ids.length === 0) return;

  try {
    if (window.go?.desktop?.App?.BatchMarkRead) {
      await window.go.desktop.App.BatchMarkRead(ids, true);
      info(`Marked ${ids.length} emails as read`);

      emails.update(list => list.map(e =>
        get(selectedIds).has(e.id) ? { ...e, isRead: true } : e
      ));
      clearSelection();
      exitSelectionMode();
    }
  } catch (err) {
    warn(`Failed to mark as read: ${err}`);
  }
}

export async function batchMarkUnread() {
  const ids = Array.from(get(selectedIds));
  if (ids.length === 0) return;

  try {
    if (window.go?.desktop?.App?.BatchMarkRead) {
      await window.go.desktop.App.BatchMarkRead(ids, false);
      info(`Marked ${ids.length} emails as unread`);

      emails.update(list => list.map(e =>
        get(selectedIds).has(e.id) ? { ...e, isRead: false } : e
      ));
      clearSelection();
      exitSelectionMode();
    }
  } catch (err) {
    warn(`Failed to mark as unread: ${err}`);
  }
}

export async function batchStar() {
  const ids = Array.from(get(selectedIds));
  if (ids.length === 0) return;

  try {
    if (window.go?.desktop?.App?.BatchStar) {
      await window.go.desktop.App.BatchStar(ids, true);
      info(`Starred ${ids.length} emails`);

      emails.update(list => list.map(e =>
        get(selectedIds).has(e.id) ? { ...e, isStarred: true } : e
      ));
    }
  } catch (err) {
    warn(`Failed to star: ${err}`);
  }
}

export async function batchUnstar() {
  const ids = Array.from(get(selectedIds));
  if (ids.length === 0) return;

  try {
    if (window.go?.desktop?.App?.BatchStar) {
      await window.go.desktop.App.BatchStar(ids, false);
      info(`Unstarred ${ids.length} emails`);

      emails.update(list => list.map(e =>
        get(selectedIds).has(e.id) ? { ...e, isStarred: false } : e
      ));
    }
  } catch (err) {
    warn(`Failed to unstar: ${err}`);
  }
}
