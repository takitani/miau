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
    var newIds = new Set(ids);
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
  var fromIndex = get(lastSelectedIndex);
  if (fromIndex < 0) fromIndex = 0;

  var $emails = get(emails);
  var start = Math.min(fromIndex, toIndex);
  var end = Math.max(fromIndex, toIndex);

  selectedIds.update(ids => {
    var newIds = new Set(ids);
    for (var i = start; i <= end; i++) {
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
  var $emails = get(emails);
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
  var $emails = get(emails);
  var $selectedIds = get(selectedIds);

  var newIds = new Set();
  for (var e of $emails) {
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
  var $emails = get(emails);
  var matching = $emails.filter(e => e.fromEmail === senderEmail);

  selectedIds.update(ids => {
    var newIds = new Set(ids);
    for (var e of matching) {
      newIds.add(e.id);
    }
    return newIds;
  });

  selectionMode.set(true);
  info(`Selected ${matching.length} emails from ${senderEmail}`);
}

// Smart selection: unread
export function selectUnread() {
  var $emails = get(emails);
  var unread = $emails.filter(e => !e.isRead);

  selectedIds.set(new Set(unread.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${unread.length} unread emails`);
}

// Smart selection: read
export function selectRead() {
  var $emails = get(emails);
  var read = $emails.filter(e => e.isRead);

  selectedIds.set(new Set(read.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${read.length} read emails`);
}

// Smart selection: with attachments
export function selectWithAttachments() {
  var $emails = get(emails);
  var withAtt = $emails.filter(e => e.hasAttachments);

  selectedIds.set(new Set(withAtt.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${withAtt.length} emails with attachments`);
}

// Smart selection: starred
export function selectStarred() {
  var $emails = get(emails);
  var starred = $emails.filter(e => e.isStarred);

  selectedIds.set(new Set(starred.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${starred.length} starred emails`);
}

// Smart selection: today
export function selectToday() {
  var $emails = get(emails);
  var today = new Date();
  today.setHours(0, 0, 0, 0);

  var todayEmails = $emails.filter(e => {
    var emailDate = new Date(e.date);
    emailDate.setHours(0, 0, 0, 0);
    return emailDate.getTime() === today.getTime();
  });

  selectedIds.set(new Set(todayEmails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${todayEmails.length} emails from today`);
}

// Smart selection: this week
export function selectThisWeek() {
  var $emails = get(emails);
  var now = new Date();
  var weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

  var weekEmails = $emails.filter(e => new Date(e.date) >= weekAgo);

  selectedIds.set(new Set(weekEmails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${weekEmails.length} emails from this week`);
}

// Smart selection: older than a week
export function selectOlderThanWeek() {
  var $emails = get(emails);
  var now = new Date();
  var weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

  var oldEmails = $emails.filter(e => new Date(e.date) < weekAgo);

  selectedIds.set(new Set(oldEmails.map(e => e.id)));
  selectionMode.set(true);
  info(`Selected ${oldEmails.length} emails older than a week`);
}

// Drag selection helpers
export function startDragSelection(index) {
  dragSelecting.set(true);
  dragStartIndex.set(index);
  var $emails = get(emails);
  if ($emails[index]) {
    selectedIds.set(new Set([$emails[index].id]));
  }
  selectionMode.set(true);
}

export function updateDragSelection(currentIndex) {
  if (!get(dragSelecting)) return;

  var startIdx = get(dragStartIndex);
  var $emails = get(emails);
  var start = Math.min(startIdx, currentIndex);
  var end = Math.max(startIdx, currentIndex);

  var newIds = new Set();
  for (var i = start; i <= end; i++) {
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
  var ids = Array.from(get(selectedIds));
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
  var ids = Array.from(get(selectedIds));
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
  var ids = Array.from(get(selectedIds));
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
  var ids = Array.from(get(selectedIds));
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
  var ids = Array.from(get(selectedIds));
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
  var ids = Array.from(get(selectedIds));
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
