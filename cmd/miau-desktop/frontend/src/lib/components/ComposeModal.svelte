<script>
  import { onMount } from 'svelte';
  import { showCompose } from '../stores/ui.js';
  import { info, error as logError } from '../stores/debug.js';

  // Form fields
  let to = '';
  let cc = '';
  let bcc = '';
  let subject = '';
  let body = '';
  let isHtml = false;
  let replyToId = null;

  // UI state
  let sending = false;
  let showCcBcc = false;
  let signature = '';
  let toInput;

  // Mode: 'new', 'reply', 'replyAll', 'forward'
  let mode = 'new';

  async function loadSignature() {
    try {
      if (window.go?.desktop?.App) {
        var sig = await window.go.desktop.App.GetSignature();
        if (sig && !body.includes(sig)) {
          signature = sig;
          body = body ? `\n\n${sig}\n\n${body}` : `\n\n${sig}`;
        }
      }
    } catch (err) {
      console.error('Failed to load signature:', err);
    }
  }

  onMount(() => {
    // DISABLED: signature loading causes signal 11 crash due to Go 1.25/WebKit GTK conflict
    // Fixed by pre-loading signature in backend
    loadSignature();

    // Check for compose context (reply, forward, etc)
    if (window.composeContext) {
      var ctx = window.composeContext;
      mode = ctx.mode || 'new';

      if (mode === 'reply' && ctx.replyTo) {
        var email = ctx.replyTo;
        to = email.fromEmail || '';
        subject = email.subject?.startsWith('Re:') ? email.subject : `Re: ${email.subject || ''}`;
        replyToId = email.id;
        body = buildQuotedBody(email);
      } else if (mode === 'replyAll' && ctx.replyTo) {
        var email = ctx.replyTo;
        to = email.fromEmail || '';
        // Add other recipients to CC
        var allRecipients = [];
        if (email.toAddresses) {
          allRecipients.push(...email.toAddresses.split(',').map(s => s.trim()));
        }
        if (email.ccAddresses) {
          allRecipients.push(...email.ccAddresses.split(',').map(s => s.trim()));
        }
        // Remove self from recipients (would need current user email)
        cc = allRecipients.filter(r => r !== to).join(', ');
        if (cc) showCcBcc = true;
        subject = email.subject?.startsWith('Re:') ? email.subject : `Re: ${email.subject || ''}`;
        replyToId = email.id;
        body = buildQuotedBody(email);
      } else if (mode === 'forward' && ctx.forwardEmail) {
        var email = ctx.forwardEmail;
        subject = email.subject?.startsWith('Fwd:') ? email.subject : `Fwd: ${email.subject || ''}`;
        body = buildForwardBody(email);
      }

      window.composeContext = null;
    }

    // Focus to field
    setTimeout(() => {
      if (toInput) toInput.focus();
    }, 100);
  });

  function buildQuotedBody(email) {
    var date = new Date(email.date).toLocaleString('pt-BR');
    var from = email.fromName ? `${email.fromName} <${email.fromEmail}>` : email.fromEmail;
    var quoted = email.bodyText || stripHtml(email.bodyHtml) || email.snippet || '';
    var lines = quoted.split('\n').map(line => `> ${line}`).join('\n');
    return `\n\nEm ${date}, ${from} escreveu:\n\n${lines}`;
  }

  function buildForwardBody(email) {
    var date = new Date(email.date).toLocaleString('pt-BR');
    var from = email.fromName ? `${email.fromName} <${email.fromEmail}>` : email.fromEmail;
    var content = email.bodyText || stripHtml(email.bodyHtml) || email.snippet || '';
    return `\n\n---------- Mensagem encaminhada ----------\nDe: ${from}\nData: ${date}\nAssunto: ${email.subject}\n\n${content}`;
  }

  function stripHtml(html) {
    if (!html) return '';
    return html.replace(/<[^>]*>/g, '').replace(/&nbsp;/g, ' ').trim();
  }

  function close() {
    showCompose.set(false);
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      send();
    } else if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'd') {
      e.preventDefault();
      saveDraft();
    }
  }

  async function send() {
    if (!to.trim()) {
      alert('Por favor, informe o destinatario');
      return;
    }

    sending = true;
    info(`Enviando email para ${to}...`);

    try {
      if (window.go?.desktop?.App) {
        var request = {
          to: to.split(',').map(s => s.trim()).filter(Boolean),
          cc: cc ? cc.split(',').map(s => s.trim()).filter(Boolean) : [],
          bcc: bcc ? bcc.split(',').map(s => s.trim()).filter(Boolean) : [],
          subject: subject,
          body: body,
          isHtml: isHtml,
          replyTo: replyToId || 0
        };

        var result = await window.go.desktop.App.SendEmail(request);

        if (result.success) {
          info(`Email enviado! MessageID: ${result.messageId}`);
          close();
        } else {
          logError('Falha ao enviar', result.error);
          alert(`Erro: ${result.error}`);
        }
      } else {
        // Mock for dev
        info('[Mock] Email enviado com sucesso');
        close();
      }
    } catch (err) {
      logError('Erro ao enviar email', err);
      alert(`Erro: ${err.message}`);
    } finally {
      sending = false;
    }
  }

  async function saveDraft() {
    info('Salvando rascunho...');
    try {
      if (window.go?.desktop?.App) {
        var draft = {
          id: 0,
          to: to.split(',').map(s => s.trim()).filter(Boolean),
          cc: cc ? cc.split(',').map(s => s.trim()).filter(Boolean) : [],
          bcc: bcc ? bcc.split(',').map(s => s.trim()).filter(Boolean) : [],
          subject: subject,
          bodyHtml: isHtml ? body : '',
          bodyText: isHtml ? '' : body,
          replyToId: replyToId || 0
        };

        var id = await window.go.desktop.App.SaveDraft(draft);
        info(`Rascunho salvo (ID: ${id})`);
      } else {
        info('[Mock] Rascunho salvo');
      }
    } catch (err) {
      logError('Erro ao salvar rascunho', err);
    }
  }

  function getTitle() {
    switch (mode) {
      case 'reply': return 'Responder';
      case 'replyAll': return 'Responder a Todos';
      case 'forward': return 'Encaminhar';
      default: return 'Novo Email';
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="overlay" on:click={close} role="button" tabindex="-1">
  <div class="compose-modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="compose-header">
      <h2>{getTitle()}</h2>
      <button class="close-btn" on:click={close}>✕</button>
    </div>

    <div class="compose-form">
      <!-- To -->
      <div class="field">
        <label for="to">Para:</label>
        <input
          bind:this={toInput}
          id="to"
          type="text"
          bind:value={to}
          placeholder="email@exemplo.com"
        />
      </div>

      <!-- CC/BCC toggle -->
      {#if !showCcBcc}
        <button class="link-btn" on:click={() => showCcBcc = true}>
          + Cc/Bcc
        </button>
      {/if}

      <!-- CC -->
      {#if showCcBcc}
        <div class="field">
          <label for="cc">Cc:</label>
          <input
            id="cc"
            type="text"
            bind:value={cc}
            placeholder="email@exemplo.com"
          />
        </div>

        <!-- BCC -->
        <div class="field">
          <label for="bcc">Bcc:</label>
          <input
            id="bcc"
            type="text"
            bind:value={bcc}
            placeholder="email@exemplo.com"
          />
        </div>
      {/if}

      <!-- Subject -->
      <div class="field">
        <label for="subject">Assunto:</label>
        <input
          id="subject"
          type="text"
          bind:value={subject}
          placeholder="Assunto do email"
        />
      </div>

      <!-- Body -->
      <div class="field body-field">
        <textarea
          bind:value={body}
          placeholder="Escreva sua mensagem..."
          rows="12"
        ></textarea>
      </div>
    </div>

    <div class="compose-footer">
      <div class="footer-left">
        <button class="send-btn" on:click={send} disabled={sending}>
          {sending ? 'Enviando...' : 'Enviar'}
        </button>
        <button class="draft-btn" on:click={saveDraft} disabled={sending}>
          Salvar Rascunho
        </button>
      </div>
      <div class="footer-right">
        <span class="hint">
          <kbd>Ctrl+Enter</kbd> enviar
          <kbd>Ctrl+Shift+D</kbd> rascunho
          <kbd>Esc</kbd> fechar
        </span>
      </div>
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

  .compose-modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    width: 90%;
    max-width: 700px;
    height: 80vh;
    max-height: 800px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .compose-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .compose-header h2 {
    margin: 0;
    font-size: var(--font-lg);
    font-weight: 600;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: var(--radius-sm);
  }

  .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .compose-form {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-md) var(--space-lg);
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }

  .field {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .field label {
    min-width: 60px;
    font-size: var(--font-sm);
    color: var(--text-muted);
    text-align: right;
  }

  .field input {
    flex: 1;
    padding: var(--space-sm);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: var(--font-sm);
  }

  .field input:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  .link-btn {
    align-self: flex-start;
    margin-left: 68px;
    background: transparent;
    border: none;
    color: var(--accent-primary);
    font-size: var(--font-sm);
    cursor: pointer;
    padding: 2px 0;
  }

  .link-btn:hover {
    text-decoration: underline;
  }

  .body-field {
    flex: 1;
    align-items: flex-start;
    min-height: 200px;
  }

  .body-field textarea {
    flex: 1;
    width: 100%;
    height: 100%;
    min-height: 200px;
    padding: var(--space-sm);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-primary);
    font-size: var(--font-sm);
    font-family: inherit;
    resize: none;
    line-height: 1.6;
  }

  .body-field textarea:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  .compose-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-top: 1px solid var(--border-color);
    background: var(--bg-tertiary);
  }

  .footer-left {
    display: flex;
    gap: var(--space-sm);
  }

  .send-btn {
    padding: var(--space-sm) var(--space-lg);
    background: var(--accent-primary);
    border: none;
    border-radius: var(--radius-md);
    color: white;
    font-size: var(--font-sm);
    font-weight: 600;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .send-btn:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  .send-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .draft-btn {
    padding: var(--space-sm) var(--space-md);
    background: transparent;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    color: var(--text-secondary);
    font-size: var(--font-sm);
    cursor: pointer;
  }

  .draft-btn:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .footer-right {
    display: flex;
    align-items: center;
  }

  .hint {
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .hint kbd {
    padding: 1px 4px;
    background: var(--bg-primary);
    border-radius: 3px;
    font-family: monospace;
    margin: 0 4px;
  }
</style>
