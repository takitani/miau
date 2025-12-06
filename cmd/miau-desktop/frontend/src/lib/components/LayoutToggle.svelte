<script>
  import { layoutMode, toggleLayoutMode } from '../stores/layout.js';

  $: isModern = $layoutMode === 'modern';
</script>

<button
  class="layout-toggle"
  class:modern={isModern}
  on:click={toggleLayoutMode}
  title="Alternar layout (L)"
>
  <span class="toggle-track">
    <span class="toggle-icon legacy">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="7" height="18" rx="1"/>
        <rect x="14" y="3" width="7" height="18" rx="1"/>
      </svg>
    </span>
    <span class="toggle-icon modern">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="3" width="5" height="18" rx="1"/>
        <rect x="11" y="3" width="10" height="8" rx="1"/>
        <rect x="11" y="13" width="10" height="8" rx="1"/>
      </svg>
    </span>
    <span class="toggle-thumb"></span>
  </span>
  <span class="toggle-label">{isModern ? 'Modern' : 'Legacy'}</span>
</button>

<style>
  .layout-toggle {
    position: fixed;
    bottom: 40px; /* Above status bar */
    right: 16px;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-radius: var(--radius-md);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    cursor: pointer;
    transition: all var(--transition-fast);
    box-shadow: var(--shadow-md);
    opacity: 0.75;
  }

  .layout-toggle:hover {
    background: var(--bg-hover);
    border-color: var(--accent-primary);
    opacity: 1;
    transform: translateY(-2px);
    box-shadow: var(--shadow-lg);
  }

  .toggle-track {
    position: relative;
    width: 48px;
    height: 24px;
    background: var(--bg-tertiary);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 4px;
  }

  .toggle-icon {
    width: 16px;
    height: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
    transition: color var(--transition-fast);
    z-index: 1;
  }

  .layout-toggle:not(.modern) .toggle-icon.legacy {
    color: var(--accent-primary);
  }

  .layout-toggle.modern .toggle-icon.modern {
    color: var(--accent-primary);
  }

  .toggle-thumb {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 20px;
    height: 20px;
    background: var(--accent-primary);
    border-radius: 50%;
    transition: transform var(--transition-normal);
    box-shadow: var(--shadow-sm);
  }

  .layout-toggle.modern .toggle-thumb {
    transform: translateX(24px);
  }

  .toggle-label {
    font-size: var(--font-xs);
    font-weight: 500;
    color: var(--text-secondary);
    min-width: 48px;
  }
</style>
