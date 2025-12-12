<script>
  import { theme, themeLabels } from '../stores/theme.js';

  function cycleTheme() {
    theme.toggle();
  }
</script>

<button
  class="theme-toggle"
  on:click={cycleTheme}
  title="Tema: {themeLabels[$theme]} (clique para alternar)"
>
  {#if $theme === 'light'}
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="5"/>
      <line x1="12" y1="1" x2="12" y2="3"/>
      <line x1="12" y1="21" x2="12" y2="23"/>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
      <line x1="1" y1="12" x2="3" y2="12"/>
      <line x1="21" y1="12" x2="23" y2="12"/>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
    </svg>
  {:else if $theme === 'dark'}
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/>
    </svg>
  {:else}
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="5"/>
      <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
      <path d="M12 7a5 5 0 010 10" fill="currentColor" opacity="0.3"/>
    </svg>
  {/if}
  <span class="theme-label">{themeLabels[$theme]}</span>
</button>

<style>
  .theme-toggle {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-xs) var(--space-sm);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    font-size: var(--font-sm);
    transition: all var(--transition-fast);
  }

  .theme-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .theme-label {
    font-size: var(--font-xs);
    font-weight: var(--weight-medium);
  }

  @media (max-width: 600px) {
    .theme-label {
      display: none;
    }
  }
</style>
