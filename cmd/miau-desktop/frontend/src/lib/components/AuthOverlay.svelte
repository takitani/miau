<script>
  import { createEventDispatcher } from 'svelte';

  export var inProgress = false;
  export var error = null;

  var dispatch = createEventDispatcher();

  function handleRetry() {
    dispatch('retry');
  }
</script>

<div class="auth-overlay">
  <div class="auth-modal">
    <div class="auth-icon">
      {#if inProgress}
        <svg class="spinner" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12a9 9 0 11-9-9"/>
        </svg>
      {:else if error}
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
      {:else}
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
          <path d="M7 11V7a5 5 0 0110 0v4"/>
        </svg>
      {/if}
    </div>

    <h2 class="auth-title">
      {#if inProgress}
        Authenticating...
      {:else if error}
        Authentication Failed
      {:else}
        Authentication Required
      {/if}
    </h2>

    <p class="auth-message">
      {#if inProgress}
        Please complete the authentication in your browser.
        <br/>
        <span class="hint">A browser window should have opened automatically.</span>
      {:else if error}
        {error}
      {:else}
        You need to authenticate with Google to continue.
      {/if}
    </p>

    {#if error}
      <button class="retry-btn" on:click={handleRetry}>
        Try Again
      </button>
    {/if}

    {#if inProgress}
      <div class="progress-indicator">
        <div class="progress-dot"></div>
        <div class="progress-dot"></div>
        <div class="progress-dot"></div>
      </div>
    {/if}
  </div>
</div>

<style>
  .auth-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    backdrop-filter: blur(8px);
  }

  .auth-modal {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    padding: 48px 64px;
    text-align: center;
    min-width: 480px;
    max-width: 560px;
    border: 1px solid var(--border-color);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .auth-icon {
    margin-bottom: var(--space-lg);
    color: var(--accent-primary);
  }

  .auth-icon svg.spinner {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .auth-title {
    font-size: var(--font-xl);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 var(--space-md) 0;
  }

  .auth-message {
    font-size: var(--font-md);
    color: var(--text-secondary);
    margin: 0 0 var(--space-lg) 0;
    line-height: 1.6;
  }

  .auth-message .hint {
    font-size: var(--font-sm);
    color: var(--text-muted);
    display: block;
    margin-top: var(--space-sm);
  }

  .retry-btn {
    padding: var(--space-sm) var(--space-lg);
    background: var(--accent-primary);
    color: var(--bg-primary);
    border: none;
    border-radius: var(--radius-sm);
    font-size: var(--font-md);
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .retry-btn:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }

  .progress-indicator {
    display: flex;
    justify-content: center;
    gap: var(--space-sm);
    margin-top: var(--space-lg);
  }

  .progress-dot {
    width: 8px;
    height: 8px;
    background: var(--accent-primary);
    border-radius: 50%;
    animation: pulse 1.4s ease-in-out infinite;
  }

  .progress-dot:nth-child(2) {
    animation-delay: 0.2s;
  }

  .progress-dot:nth-child(3) {
    animation-delay: 0.4s;
  }

  @keyframes pulse {
    0%, 80%, 100% {
      opacity: 0.3;
      transform: scale(0.8);
    }
    40% {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
