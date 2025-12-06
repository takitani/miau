import { writable, get } from 'svelte/store';
import { info, warn } from './debug.js';

// Contacts state
export const contacts = writable([]);
export const topContacts = writable([]);
export const contactsLoading = writable(false);
export const contactsSyncing = writable(false);
export const syncStatus = writable(null);

// Search contacts
export async function searchContacts(query, limit = 20) {
  if (!query || query.length < 2) {
    return [];
  }

  try {
    if (window.go?.desktop?.App?.SearchContacts) {
      var results = await window.go.desktop.App.SearchContacts(query, limit);
      return results || [];
    }
  } catch (err) {
    warn(`Failed to search contacts: ${err}`);
  }

  return [];
}

// Load top contacts (most frequently contacted)
export async function loadTopContacts(limit = 10) {
  contactsLoading.set(true);
  try {
    if (window.go?.desktop?.App?.GetTopContacts) {
      var results = await window.go.desktop.App.GetTopContacts(limit);
      topContacts.set(results || []);
      info(`Loaded ${(results || []).length} top contacts`);
    }
  } catch (err) {
    warn(`Failed to load top contacts: ${err}`);
  } finally {
    contactsLoading.set(false);
  }
}

// Sync contacts from Gmail
export async function syncContacts(fullSync = false) {
  contactsSyncing.set(true);
  try {
    if (window.go?.desktop?.App?.SyncContacts) {
      await window.go.desktop.App.SyncContacts(fullSync);
      info(`Contacts sync ${fullSync ? '(full)' : '(incremental)'} completed`);
      // Reload top contacts after sync
      await loadTopContacts();
      await loadSyncStatus();
    }
  } catch (err) {
    warn(`Failed to sync contacts: ${err}`);
  } finally {
    contactsSyncing.set(false);
  }
}

// Load sync status
export async function loadSyncStatus() {
  try {
    if (window.go?.desktop?.App?.GetContactSyncStatus) {
      var status = await window.go.desktop.App.GetContactSyncStatus();
      syncStatus.set(status);
    }
  } catch (err) {
    warn(`Failed to load sync status: ${err}`);
  }
}

// Get contact display info (for autocomplete)
export function formatContact(contact) {
  if (!contact) return '';

  var email = contact.emails?.[0]?.email || '';
  var name = contact.displayName || '';

  if (name && email) {
    return `${name} <${email}>`;
  }
  return email || name;
}

// Get primary email from contact
export function getPrimaryEmail(contact) {
  if (!contact?.emails?.length) return null;

  var primary = contact.emails.find(e => e.isPrimary);
  return primary || contact.emails[0];
}
