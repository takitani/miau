<script>
  import { onMount } from 'svelte';
  import { showAI, aiProvider, aiProviders } from '../stores/ui.js';
  import { selectedEmail } from '../stores/emails.js';
  import { info, error as logError } from '../stores/debug.js';

  export let emailContext = null;

  let input = '';
  let messages = [];
  let loading = false;
  let inputEl;

  $: currentProvider = $aiProviders.find(p => p.id === $aiProvider) || $aiProviders[0];

  onMount(() => {
    if (inputEl) inputEl.focus();

    // Add initial context message if email selected
    if (emailContext) {
      messages = [{
        role: 'system',
        content: `Contexto: Email de ${emailContext.fromName} <${emailContext.fromEmail}>\nAssunto: ${emailContext.subject}\n\n${emailContext.snippet || ''}`
      }];
    }
  });

  function close() {
    showAI.set(false);
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    } else if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  function cycleProvider() {
    const idx = $aiProviders.findIndex(p => p.id === $aiProvider);
    const nextIdx = (idx + 1) % $aiProviders.length;
    aiProvider.set($aiProviders[nextIdx].id);
    info(`AI Provider: ${$aiProviders[nextIdx].name}`);
  }

  async function sendMessage() {
    if (!input.trim() || loading) return;

    const userMessage = input.trim();
    input = '';
    loading = true;

    messages = [...messages, { role: 'user', content: userMessage }];

    try {
      info(`Sending to ${currentProvider.name}: ${userMessage.substring(0, 50)}...`);

      if (window.go?.desktop?.App) {
        const response = await window.go.desktop.App.AskAI(
          currentProvider.id,
          userMessage,
          emailContext ? JSON.stringify(emailContext) : ''
        );

        messages = [...messages, { role: 'assistant', content: response }];
        info(`AI Response received (${response.length} chars)`);
      } else {
        // Mock response for dev
        messages = [...messages, {
          role: 'assistant',
          content: `[Mock] Resposta do ${currentProvider.name} para: "${userMessage}"`
        }];
      }
    } catch (err) {
      logError('AI Error', err);
      messages = [...messages, {
        role: 'error',
        content: `Erro: ${err.message || 'Falha ao comunicar com IA'}`
      }];
    } finally {
      loading = false;
      if (inputEl) inputEl.focus();
    }
  }

  function clearChat() {
    messages = emailContext ? [{
      role: 'system',
      content: `Contexto: Email de ${emailContext.fromName} <${emailContext.fromEmail}>\nAssunto: ${emailContext.subject}`
    }] : [];
  }

  function copyLastResponse() {
    const lastAssistant = [...messages].reverse().find(m => m.role === 'assistant');
    if (lastAssistant) {
      navigator.clipboard.writeText(lastAssistant.content);
      info('Resposta copiada!');
    }
  }
</script>

<div class="overlay" on:click={close} on:keydown={handleKeydown} role="button" tabindex="-1">
  <div class="ai-modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="ai-header">
      <div class="header-left">
        <h2>Assistente IA</h2>
        <button class="provider-btn" on:click={cycleProvider} title="Trocar provider (Tab)">
          <span class="provider-icon">{currentProvider.icon}</span>
          <span class="provider-name">{currentProvider.name}</span>
        </button>
      </div>
      <div class="header-right">
        <button class="action-btn" on:click={copyLastResponse} title="Copiar ultima resposta">
          📋
        </button>
        <button class="action-btn" on:click={clearChat} title="Limpar chat">
          🗑️
        </button>
        <button class="close-btn" on:click={close}>✕</button>
      </div>
    </div>

    {#if emailContext}
      <div class="context-banner">
        <span class="context-icon">📧</span>
        <span class="context-text">
          <strong>{emailContext.subject}</strong>
          <span class="from">de {emailContext.fromName}</span>
        </span>
      </div>
    {/if}

    <div class="messages">
      {#each messages as msg}
        <div class="message {msg.role}">
          {#if msg.role === 'user'}
            <div class="message-header">Voce</div>
          {:else if msg.role === 'assistant'}
            <div class="message-header">{currentProvider.icon} {currentProvider.name}</div>
          {:else if msg.role === 'system'}
            <div class="message-header">📎 Contexto</div>
          {:else if msg.role === 'error'}
            <div class="message-header">⚠️ Erro</div>
          {/if}
          <div class="message-content">{msg.content}</div>
        </div>
      {/each}

      {#if loading}
        <div class="message assistant loading">
          <div class="message-header">{currentProvider.icon} {currentProvider.name}</div>
          <div class="message-content">
            <span class="typing-indicator">
              <span></span><span></span><span></span>
            </span>
          </div>
        </div>
      {/if}

      {#if messages.length === 0 && !loading}
        <div class="empty-state">
          <p>Pergunte qualquer coisa sobre seus emails.</p>
          <p class="examples">
            Exemplos:<br>
            "Resuma este email"<br>
            "Escreva uma resposta formal"<br>
            "Quais sao os pontos principais?"
          </p>
        </div>
      {/if}
    </div>

    <div class="input-area">
      <textarea
        bind:this={inputEl}
        bind:value={input}
        on:keydown={handleKeydown}
        placeholder="Digite sua mensagem... (Enter para enviar, Shift+Enter para nova linha)"
        rows="2"
        disabled={loading}
      ></textarea>
      <button class="send-btn" on:click={sendMessage} disabled={loading || !input.trim()}>
        {loading ? '...' : '➤'}
      </button>
    </div>

    <div class="ai-footer">
      <span class="hint">
        <kbd>Tab</kbd> trocar IA
        <kbd>Enter</kbd> enviar
        <kbd>Esc</kbd> fechar
      </span>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .ai-modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    width: 90%;
    max-width: 600px;
    height: 80vh;
    max-height: 700px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .ai-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-sm) var(--space-md);
    border-bottom: 1px solid var(--border-color);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--space-md);
  }

  .ai-header h2 {
    margin: 0;
    font-size: var(--font-md);
    font-weight: 600;
  }

  .provider-btn {
    display: flex;
    align-items: center;
    gap: var(--space-xs);
    padding: 4px 10px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: var(--font-sm);
    transition: all var(--transition-fast);
  }

  .provider-btn:hover {
    background: var(--bg-hover);
    border-color: var(--accent-primary);
  }

  .provider-icon {
    font-size: 16px;
  }

  .header-right {
    display: flex;
    gap: var(--space-xs);
  }

  .action-btn, .close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 16px;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: var(--radius-sm);
  }

  .action-btn:hover, .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .context-banner {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-tertiary);
    border-bottom: 1px solid var(--border-color);
    font-size: var(--font-sm);
  }

  .context-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .context-text .from {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .messages {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-md);
    display: flex;
    flex-direction: column;
    gap: var(--space-md);
  }

  .message {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .message-header {
    font-size: var(--font-xs);
    font-weight: 600;
    color: var(--text-muted);
  }

  .message-content {
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-md);
    font-size: var(--font-sm);
    line-height: 1.5;
    white-space: pre-wrap;
  }

  .message.user .message-content {
    background: var(--accent-primary);
    color: white;
    align-self: flex-end;
    max-width: 85%;
  }

  .message.assistant .message-content {
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
  }

  .message.system .message-content {
    background: rgba(59, 130, 246, 0.1);
    border: 1px solid rgba(59, 130, 246, 0.3);
    font-size: var(--font-xs);
    color: var(--text-secondary);
  }

  .message.error .message-content {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    color: #f87171;
  }

  .typing-indicator {
    display: flex;
    gap: 4px;
  }

  .typing-indicator span {
    width: 8px;
    height: 8px;
    background: var(--text-muted);
    border-radius: 50%;
    animation: bounce 1.4s infinite ease-in-out both;
  }

  .typing-indicator span:nth-child(1) { animation-delay: -0.32s; }
  .typing-indicator span:nth-child(2) { animation-delay: -0.16s; }

  @keyframes bounce {
    0%, 80%, 100% { transform: scale(0); }
    40% { transform: scale(1); }
  }

  .empty-state {
    text-align: center;
    color: var(--text-muted);
    padding: var(--space-xl);
  }

  .empty-state .examples {
    margin-top: var(--space-md);
    font-size: var(--font-sm);
    opacity: 0.7;
  }

  .input-area {
    display: flex;
    gap: var(--space-sm);
    padding: var(--space-md);
    border-top: 1px solid var(--border-color);
  }

  .input-area textarea {
    flex: 1;
    padding: var(--space-sm);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    color: var(--text-primary);
    font-size: var(--font-sm);
    resize: none;
    font-family: inherit;
  }

  .input-area textarea:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  .send-btn {
    padding: var(--space-sm) var(--space-md);
    background: var(--accent-primary);
    border: none;
    border-radius: var(--radius-md);
    color: white;
    font-size: 18px;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .send-btn:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  .send-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .ai-footer {
    padding: var(--space-xs) var(--space-md);
    border-top: 1px solid var(--border-color);
    text-align: center;
  }

  .hint {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .hint kbd {
    padding: 1px 4px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    font-family: monospace;
    margin: 0 4px;
  }
</style>
