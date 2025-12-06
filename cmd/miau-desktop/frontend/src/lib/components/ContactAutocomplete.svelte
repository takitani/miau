<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { searchContacts, topContacts, loadTopContacts, formatContact, getPrimaryEmail } from '../stores/contacts.js';

  export var value = '';
  export var placeholder = 'Digite um email ou nome...';
  export var id = '';
  export var disabled = false;

  var dispatch = createEventDispatcher();
  var inputEl;
  var suggestions = [];
  var showSuggestions = false;
  var selectedIndex = -1;
  var searchTimeout = null;
  var isSearching = false;
  var isSelecting = false; // Flag to prevent blur from closing suggestions

  // Parse current input to get the last token being typed
  function getLastToken() {
    var parts = value.split(/[,;]\s*/);
    return parts[parts.length - 1].trim();
  }

  // Handle input changes
  async function handleInput() {
    var token = getLastToken();

    if (token.length < 2) {
      suggestions = [];
      showSuggestions = false;
      return;
    }

    // Debounce search
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(async () => {
      isSearching = true;
      try {
        var results = await searchContacts(token, 8);
        suggestions = results || [];
        showSuggestions = suggestions.length > 0;
        selectedIndex = -1;
      } finally {
        isSearching = false;
      }
    }, 150);
  }

  // Handle keyboard navigation
  function handleKeydown(e) {
    if (!showSuggestions) {
      if (e.key === 'ArrowDown' && $topContacts.length > 0) {
        // Show top contacts on arrow down when no suggestions
        suggestions = $topContacts.slice(0, 6);
        showSuggestions = true;
        selectedIndex = 0;
        e.preventDefault();
      }
      return;
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        selectedIndex = Math.min(selectedIndex + 1, suggestions.length - 1);
        break;
      case 'ArrowUp':
        e.preventDefault();
        selectedIndex = Math.max(selectedIndex - 1, 0);
        break;
      case 'Enter':
      case 'Tab':
        if (selectedIndex >= 0 && suggestions[selectedIndex]) {
          e.preventDefault();
          selectContact(suggestions[selectedIndex]);
        }
        break;
      case 'Escape':
        e.preventDefault();
        showSuggestions = false;
        break;
    }
  }

  // Select a contact from suggestions
  function selectContact(contact) {
    var emailObj = getPrimaryEmail(contact);
    if (!emailObj || !emailObj.email) return;

    var emailStr = contact.displayName
      ? `${contact.displayName} <${emailObj.email}>`
      : emailObj.email;

    // Replace the last token with selected contact
    var parts = value.split(/[,;]\s*/);
    parts[parts.length - 1] = emailStr;
    value = parts.join(', ');

    // Add comma for next entry
    if (!value.endsWith(', ')) {
      value += ', ';
    }

    showSuggestions = false;
    selectedIndex = -1;
    dispatch('change', { value });

    // Keep focus on input
    inputEl?.focus();
  }

  // Handle blur - close suggestions after a delay (to allow click)
  function handleBlur() {
    // Longer delay to ensure mousedown/click can complete
    setTimeout(() => {
      if (!isSelecting) {
        showSuggestions = false;
      }
      isSelecting = false;
    }, 250);
  }

  // Handle focus - show top contacts if empty
  async function handleFocus() {
    if (!value.trim() && $topContacts.length > 0) {
      suggestions = $topContacts.slice(0, 6);
      showSuggestions = true;
    }
  }

  onMount(() => {
    // Load top contacts on mount
    loadTopContacts(10);
  });
</script>

<div class="autocomplete-wrapper">
  <input
    bind:this={inputEl}
    type="text"
    {id}
    {placeholder}
    {disabled}
    bind:value
    on:input={handleInput}
    on:keydown={handleKeydown}
    on:blur={handleBlur}
    on:focus={handleFocus}
    autocomplete="off"
  />

  {#if showSuggestions && suggestions.length > 0}
    <div class="suggestions" transition:fly={{ y: -5, duration: 150 }}>
      {#each suggestions as contact, index}
        <button
          type="button"
          class="suggestion"
          class:selected={index === selectedIndex}
          on:pointerdown|preventDefault|stopPropagation={() => { isSelecting = true; selectContact(contact); }}
          on:mouseenter={() => selectedIndex = index}
        >
          <div class="contact-info">
            {#if contact.photoPath || contact.photoUrl}
              <img
                class="avatar"
                src={contact.photoPath || contact.photoUrl}
                alt=""
                on:error={(e) => e.target.style.display = 'none'}
              />
            {:else}
              <div class="avatar-placeholder">
                {(contact.displayName || contact.emails?.[0]?.email || '?')[0].toUpperCase()}
              </div>
            {/if}
            <div class="contact-text">
              <span class="name">{contact.displayName || contact.emails?.[0]?.email || 'Sem nome'}</span>
              {#if contact.displayName && contact.emails?.[0]?.email}
                <span class="email">{contact.emails[0].email}</span>
              {/if}
            </div>
          </div>
          {#if contact.interactionCount > 0}
            <span class="interaction-badge" title="Interacoes">
              {contact.interactionCount}
            </span>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .autocomplete-wrapper {
    position: relative;
    width: 100%;
  }

  input {
    width: 100%;
    padding: 8px 12px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: var(--font-sm);
    outline: none;
    transition: border-color 0.15s ease;
  }

  input:focus {
    border-color: var(--accent-primary);
  }

  input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .suggestions {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 4px;
    background: rgba(30, 35, 45, 0.98);
    backdrop-filter: blur(12px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    overflow: hidden;
    z-index: 1000;
    max-height: 280px;
    overflow-y: auto;
  }

  .suggestion {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 10px 12px;
    background: transparent;
    border: none;
    color: var(--text-primary);
    text-align: left;
    cursor: pointer;
    transition: background 0.1s ease;
  }

  .suggestion:hover,
  .suggestion.selected {
    background: rgba(255, 255, 255, 0.08);
  }

  .contact-info {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    flex: 1;
  }

  .avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .avatar-placeholder {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    font-weight: 600;
    color: white;
    flex-shrink: 0;
  }

  .contact-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .name {
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .email {
    font-size: 11px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .interaction-badge {
    font-size: 10px;
    padding: 2px 6px;
    background: rgba(102, 126, 234, 0.2);
    color: #a5b4fc;
    border-radius: 10px;
    flex-shrink: 0;
  }
</style>
