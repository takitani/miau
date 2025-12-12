import { writable } from 'svelte/store';

// Theme options: 'light', 'dark', 'auto'
const THEME_KEY = 'miau-theme';

function createThemeStore() {
  // Load saved theme from localStorage
  const savedTheme = typeof localStorage !== 'undefined'
    ? localStorage.getItem(THEME_KEY) || 'light'
    : 'light';

  const { subscribe, set, update } = writable(savedTheme);

  return {
    subscribe,
    set: (theme) => {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(THEME_KEY, theme);
      }
      applyTheme(theme);
      set(theme);
    },
    toggle: () => {
      update(current => {
        const themes = ['light', 'dark', 'auto'];
        const currentIndex = themes.indexOf(current);
        const nextTheme = themes[(currentIndex + 1) % themes.length];
        if (typeof localStorage !== 'undefined') {
          localStorage.setItem(THEME_KEY, nextTheme);
        }
        applyTheme(nextTheme);
        return nextTheme;
      });
    },
    init: () => {
      applyTheme(savedTheme);
    }
  };
}

function applyTheme(theme) {
  if (typeof document === 'undefined') return;
  document.documentElement.setAttribute('data-theme', theme);
}

export const theme = createThemeStore();

// Theme display names
export const themeLabels = {
  light: 'Claro',
  dark: 'Escuro',
  auto: 'Automático'
};

// Theme icons (as SVG path data)
export const themeIcons = {
  light: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z',
  dark: 'M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z',
  auto: 'M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z'
};
