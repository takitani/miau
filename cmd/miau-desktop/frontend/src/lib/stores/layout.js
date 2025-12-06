import { writable, get } from 'svelte/store';

// Layout mode: 'legacy' | 'modern'
export var layoutMode = writable('legacy');

// Sidebar state
export var sidebarExpanded = writable(true);

// Constants
export var SIDEBAR_COLLAPSED_WIDTH = 56;
export var SIDEBAR_EXPANDED_WIDTH = 280;

// Storage key for persistence
var LAYOUT_STORAGE_KEY = 'miau-layout-preferences';

// Initialize from localStorage
export function initLayoutPreferences() {
  try {
    var saved = localStorage.getItem(LAYOUT_STORAGE_KEY);
    if (saved) {
      var prefs = JSON.parse(saved);
      if (prefs.layoutMode)
        layoutMode.set(prefs.layoutMode);
      if (prefs.sidebarExpanded !== undefined)
        sidebarExpanded.set(prefs.sidebarExpanded);
    }
  } catch (e) {
    console.error('Failed to load layout preferences:', e);
  }
}

// Save preferences to localStorage
export function saveLayoutPreferences() {
  try {
    localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify({
      layoutMode: get(layoutMode),
      sidebarExpanded: get(sidebarExpanded)
    }));
  } catch (e) {
    console.error('Failed to save layout preferences:', e);
  }
}

// Toggle layout mode
export function toggleLayoutMode() {
  layoutMode.update(mode => {
    var newMode = mode === 'legacy' ? 'modern' : 'legacy';
    setTimeout(saveLayoutPreferences, 0);
    return newMode;
  });
}

// Toggle sidebar expansion
export function toggleSidebar() {
  sidebarExpanded.update(expanded => {
    var newExpanded = !expanded;
    setTimeout(saveLayoutPreferences, 0);
    return newExpanded;
  });
}

// Get current sidebar width based on expansion state
export function getSidebarWidth(expanded) {
  return expanded ? SIDEBAR_EXPANDED_WIDTH : SIDEBAR_COLLAPSED_WIDTH;
}
