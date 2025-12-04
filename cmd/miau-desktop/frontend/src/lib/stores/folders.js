import { writable, get } from 'svelte/store';
import { currentFolder, loadEmails } from './emails.js';

// Folder list
export const folders = writable([]);

// Loading state
export const foldersLoading = writable(false);

// Load folders from backend
export async function loadFolders() {
  foldersLoading.set(true);
  try {
    if (window.go?.desktop?.App) {
      const result = await window.go.desktop.App.GetFolders();
      folders.set(result || []);
    } else {
      // Mock data for development
      folders.set(getMockFolders());
    }
  } catch (err) {
    console.error('Failed to load folders:', err);
    folders.set([]);
  } finally {
    foldersLoading.set(false);
  }
}

// Select a folder
export async function selectFolder(name) {
  console.log(`[selectFolder] START: "${name}"`);
  console.log(`[selectFolder] Window size: ${window.innerWidth}x${window.innerHeight}`);
  console.log(`[selectFolder] Document font-size: ${getComputedStyle(document.documentElement).fontSize}`);

  try {
    if (window.go?.desktop?.App) {
      console.log(`[selectFolder] Calling backend SelectFolder...`);
      await window.go.desktop.App.SelectFolder(name);
      console.log(`[selectFolder] Backend returned`);
    }
    currentFolder.set(name);
    console.log(`[selectFolder] Loading emails...`);
    await loadEmails(name);
    console.log(`[selectFolder] DONE. Document font-size: ${getComputedStyle(document.documentElement).fontSize}`);
  } catch (err) {
    console.error('Failed to select folder:', err);
  }
}

// Get folder by name
export function getFolder(name) {
  return get(folders).find(f => f.name === name);
}

// Mock data for development
function getMockFolders() {
  return [
    { id: 1, name: 'INBOX', totalMessages: 42, unreadMessages: 5 },
    { id: 2, name: '[Gmail]/Sent Mail', totalMessages: 128, unreadMessages: 0 },
    { id: 3, name: '[Gmail]/Drafts', totalMessages: 3, unreadMessages: 0 },
    { id: 4, name: '[Gmail]/Starred', totalMessages: 12, unreadMessages: 2 },
    { id: 5, name: '[Gmail]/Trash', totalMessages: 8, unreadMessages: 0 },
    { id: 6, name: '[Gmail]/All Mail', totalMessages: 1234, unreadMessages: 15 }
  ];
}
