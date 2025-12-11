<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { GetCurrentAccount, GetAllAccounts, SetCurrentAccount } from '../../../bindings/github.com/opik/miau/internal/desktop/app.js';
  import { info } from '../stores/debug.js';

  const dispatch = createEventDispatcher();

  var currentAccount = null;
  var accounts = [];
  var isOpen = false;
  var isLoading = false;

  // Get initials from email or name
  function getInitials(account) {
    if (account.name) {
      var parts = account.name.split(' ');
      if (parts.length >= 2) {
        return (parts[0][0] + parts[1][0]).toUpperCase();
      }
      return account.name.substring(0, 2).toUpperCase();
    }
    return account.email.substring(0, 2).toUpperCase();
  }

  // Get display name
  function getDisplayName(account) {
    return account.name || account.email.split('@')[0];
  }

  // Load accounts on mount
  async function loadAccounts() {
    try {
      currentAccount = await GetCurrentAccount();
      accounts = await GetAllAccounts() || [];
    } catch (err) {
      console.error('Failed to load accounts:', err);
    }
  }

  // Switch to a different account
  async function switchAccount(email) {
    if (email === currentAccount?.email) {
      isOpen = false;
      return;
    }

    isLoading = true;
    try {
      info(`Switching to account: ${email}`);
      await SetCurrentAccount(email);
      currentAccount = await GetCurrentAccount();
      isOpen = false;
      dispatch('switched', { email });
    } catch (err) {
      console.error('Failed to switch account:', err);
    } finally {
      isLoading = false;
    }
  }

  // Toggle dropdown
  function toggleDropdown() {
    if (accounts.length > 1) {
      isOpen = !isOpen;
    }
  }

  // Close on click outside
  function handleClickOutside(event) {
    if (isOpen && !event.target.closest('.account-selector')) {
      isOpen = false;
    }
  }

  onMount(() => {
    loadAccounts();
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  });
</script>

<div class="account-selector" class:has-multiple={accounts.length > 1}>
  <button
    class="current-account"
    on:click={toggleDropdown}
    disabled={isLoading || accounts.length <= 1}
    title={currentAccount ? `${currentAccount.email}${accounts.length > 1 ? ' (click to switch)' : ''}` : 'Loading...'}
  >
    {#if currentAccount}
      <div class="avatar">
        {getInitials(currentAccount)}
      </div>
      <div class="account-info">
        <span class="account-name">{getDisplayName(currentAccount)}</span>
        <span class="account-email">{currentAccount.email}</span>
      </div>
      {#if accounts.length > 1}
        <svg class="chevron" class:open={isOpen} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M6 9l6 6 6-6"/>
        </svg>
      {/if}
    {:else}
      <div class="avatar loading">...</div>
      <div class="account-info">
        <span class="account-name">Loading...</span>
      </div>
    {/if}
  </button>

  {#if isOpen && accounts.length > 1}
    <div class="dropdown">
      {#each accounts as account}
        <button
          class="account-option"
          class:active={account.email === currentAccount?.email}
          on:click={() => switchAccount(account.email)}
          disabled={isLoading}
        >
          <div class="avatar" class:active={account.email === currentAccount?.email}>
            {getInitials(account)}
          </div>
          <div class="account-info">
            <span class="account-name">{getDisplayName(account)}</span>
            <span class="account-email">{account.email}</span>
          </div>
          {#if account.email === currentAccount?.email}
            <svg class="check-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 6L9 17l-5-5"/>
            </svg>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .account-selector {
    position: relative;
    width: 100%;
  }

  .current-account {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-sm);
    border: none;
    background: transparent;
    color: var(--text-primary);
    cursor: default;
    border-radius: var(--radius-md);
    transition: all var(--transition-fast);
    text-align: left;
  }

  .has-multiple .current-account {
    cursor: pointer;
  }

  .has-multiple .current-account:hover:not(:disabled) {
    background: var(--bg-hover);
  }

  .has-multiple .current-account:disabled {
    opacity: 0.7;
    cursor: wait;
  }

  .avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--accent-primary);
    color: white;
    font-size: var(--font-sm);
    font-weight: 600;
    flex-shrink: 0;
  }

  .avatar.loading {
    background: var(--bg-tertiary);
    color: var(--text-muted);
  }

  .avatar.active {
    background: var(--accent-primary);
  }

  .account-info {
    flex: 1;
    min-width: 0;
    overflow: hidden;
  }

  .account-name {
    display: block;
    font-size: var(--font-sm);
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .account-email {
    display: block;
    font-size: var(--font-xs);
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .chevron {
    color: var(--text-muted);
    transition: transform var(--transition-fast);
    flex-shrink: 0;
  }

  .chevron.open {
    transform: rotate(180deg);
  }

  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: var(--space-xs);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    z-index: 100;
    overflow: hidden;
    animation: slideDown 150ms ease;
  }

  @keyframes slideDown {
    from {
      opacity: 0;
      transform: translateY(-8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .account-option {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-sm);
    border: none;
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    text-align: left;
    transition: all var(--transition-fast);
  }

  .account-option:hover:not(:disabled) {
    background: var(--bg-hover);
  }

  .account-option.active {
    background: var(--bg-active);
  }

  .account-option:disabled {
    opacity: 0.7;
    cursor: wait;
  }

  .account-option .avatar {
    width: 28px;
    height: 28px;
    font-size: var(--font-xs);
    background: var(--bg-tertiary);
    color: var(--text-secondary);
  }

  .account-option.active .avatar {
    background: var(--accent-primary);
    color: white;
  }

  .check-icon {
    color: var(--accent-primary);
    flex-shrink: 0;
  }
</style>
