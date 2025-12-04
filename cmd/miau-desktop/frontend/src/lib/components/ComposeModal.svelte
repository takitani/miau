<script>
  import { onMount } from 'svelte';
  import { showCompose } from '../stores/ui.js';
  import { info, error as logError } from '../stores/debug.js';

  // Form fields
  let to = '';
  let cc = '';
  let bcc = '';
  let subject = '';
  let bodyText = '';
  let bodyHtml = '';
  let isHtml = true; // Default to HTML mode
  let replyToId = null;

  // UI state
  let sending = false;
  let showCcBcc = false;
  let signature = '';
  let signatureHtml = '';
  let toInput;
  let htmlEditor;

  // Mode: 'new', 'reply', 'replyAll', 'forward'
  let mode = 'new';

  async function loadSignature() {
    try {
      if (window.go?.desktop?.App) {
        var sig = await window.go.desktop.App.GetSignature();
        if (sig) {
          signatureHtml = sig;
          signature = stripHtml(sig);
        }
      }
    } catch (err) {
      console.error('Failed to load signature:', err);
    }
  }

  onMount(async () => {
    await loadSignature();

    // Check for compose context (reply, forward, etc)
    if (window.composeContext) {
      var ctx = window.composeContext;
      mode = ctx.mode || 'new';

      if (mode === 'reply' && ctx.replyTo) {
        var email = ctx.replyTo;
        to = email.fromEmail || '';
        subject = email.subject?.startsWith('Re:') ? email.subject : `Re: ${email.subject || ''}`;
        replyToId = email.id;

        // Use HTML if original was HTML
        if (email.bodyHtml) {
          isHtml = true;
          bodyHtml = buildQuotedBodyHtml(email);
        } else {
          isHtml = false;
          bodyText = buildQuotedBodyText(email);
        }
      } else if (mode === 'replyAll' && ctx.replyTo) {
        var email = ctx.replyTo;
        to = email.fromEmail || '';
        var allRecipients = [];
        if (email.toAddresses) {
          allRecipients.push(...email.toAddresses.split(',').map(s => s.trim()));
        }
        if (email.ccAddresses) {
          allRecipients.push(...email.ccAddresses.split(',').map(s => s.trim()));
        }
        cc = allRecipients.filter(r => r !== to).join(', ');
        if (cc) showCcBcc = true;
        subject = email.subject?.startsWith('Re:') ? email.subject : `Re: ${email.subject || ''}`;
        replyToId = email.id;

        if (email.bodyHtml) {
          isHtml = true;
          bodyHtml = buildQuotedBodyHtml(email);
        } else {
          isHtml = false;
          bodyText = buildQuotedBodyText(email);
        }
      } else if (mode === 'forward' && ctx.forwardEmail) {
        var email = ctx.forwardEmail;
        subject = email.subject?.startsWith('Fwd:') ? email.subject : `Fwd: ${email.subject || ''}`;

        if (email.bodyHtml) {
          isHtml = true;
          bodyHtml = buildForwardBodyHtml(email);
        } else {
          isHtml = false;
          bodyText = buildForwardBodyText(email);
        }
      } else {
        // New email - add signature
        if (signatureHtml) {
          isHtml = true;
          bodyHtml = `<br><br>${signatureHtml}`;
        } else if (signature) {
          bodyText = `\n\n${signature}`;
        }
      }

      window.composeContext = null;
    } else {
      // New email - add signature
      if (signatureHtml) {
        isHtml = true;
        bodyHtml = `<br><br>${signatureHtml}`;
      } else if (signature) {
        bodyText = `\n\n${signature}`;
      }
    }

    // Update HTML editor content
    setTimeout(() => {
      if (htmlEditor && isHtml) {
        htmlEditor.innerHTML = bodyHtml;
      }
    }, 50);

    // Focus to field
    setTimeout(() => {
      if (toInput) toInput.focus();
    }, 100);
  });

  function buildQuotedBodyText(email) {
    var date = new Date(email.date).toLocaleString('pt-BR');
    var from = email.fromName ? `${email.fromName} <${email.fromEmail}>` : email.fromEmail;
    var quoted = email.bodyText || stripHtml(email.bodyHtml) || email.snippet || '';
    var lines = quoted.split('\n').map(line => `> ${line}`).join('\n');
    var sig = signature ? `\n\n${signature}` : '';
    return `${sig}\n\nEm ${date}, ${from} escreveu:\n\n${lines}`;
  }

  function buildQuotedBodyHtml(email) {
    var date = new Date(email.date).toLocaleString('pt-BR');
    var from = email.fromName ? `${email.fromName} &lt;${email.fromEmail}&gt;` : email.fromEmail;
    var quoted = email.bodyHtml || `<p>${escapeHtml(email.bodyText || email.snippet || '')}</p>`;
    var sig = signatureHtml ? `<br><br>${signatureHtml}` : '';
    return `${sig}<br><br><div style="border-left: 2px solid #ccc; padding-left: 10px; margin-left: 5px; color: #666;">
      <p>Em ${date}, ${from} escreveu:</p>
      ${quoted}
    </div>`;
  }

  function buildForwardBodyText(email) {
    var date = new Date(email.date).toLocaleString('pt-BR');
    var from = email.fromName ? `${email.fromName} <${email.fromEmail}>` : email.fromEmail;
    var content = email.bodyText || stripHtml(email.bodyHtml) || email.snippet || '';
    var sig = signature ? `\n\n${signature}` : '';
    return `${sig}\n\n---------- Mensagem encaminhada ----------\nDe: ${from}\nData: ${date}\nAssunto: ${email.subject}\n\n${content}`;
  }

  function buildForwardBodyHtml(email) {
    var date = new Date(email.date).toLocaleString('pt-BR');
    var from = email.fromName ? `${email.fromName} &lt;${email.fromEmail}&gt;` : email.fromEmail;
    var content = email.bodyHtml || `<p>${escapeHtml(email.bodyText || email.snippet || '')}</p>`;
    var sig = signatureHtml ? `<br><br>${signatureHtml}` : '';
    return `${sig}<br><br><hr style="border: none; border-top: 1px solid #ccc;">
      <p><strong>---------- Mensagem encaminhada ----------</strong><br>
      De: ${from}<br>
      Data: ${date}<br>
      Assunto: ${escapeHtml(email.subject || '')}</p>
      ${content}`;
  }

  function escapeHtml(text) {
    if (!text) return '';
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;')
      .replace(/\n/g, '<br>');
  }

  function stripHtml(html) {
    if (!html) return '';
    return html
      .replace(/<br\s*\/?>/gi, '\n')
      .replace(/<\/p>/gi, '\n')
      .replace(/<\/div>/gi, '\n')
      .replace(/<\/tr>/gi, '\n')
      .replace(/<\/td>/gi, ' ')
      .replace(/<\/th>/gi, ' ')
      .replace(/<\/li>/gi, '\n')
      .replace(/<[^>]*>/g, '')
      .replace(/&nbsp;/g, ' ')
      .replace(/&amp;/g, '&')
      .replace(/&lt;/g, '<')
      .replace(/&gt;/g, '>')
      .replace(/&quot;/g, '"')
      .replace(/&#39;/g, "'")
      .replace(/\n\s*\n\s*\n/g, '\n\n')
      .trim();
  }

  function toggleMode() {
    if (isHtml) {
      // Switching to text
      if (htmlEditor) {
        bodyText = stripHtml(htmlEditor.innerHTML);
      }
      isHtml = false;
    } else {
      // Switching to HTML
      bodyHtml = escapeHtml(bodyText);
      isHtml = true;
      setTimeout(() => {
        if (htmlEditor) {
          htmlEditor.innerHTML = bodyHtml;
        }
      }, 10);
    }
  }

  function close() {
    showCompose.set(false);
  }

  function handleKeydown(e) {
    // Only handle special keys, let normal typing pass through
    if (e.key === 'Escape') {
      close();
    } else if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      send();
    } else if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === 'd' || e.key === 'D')) {
      e.preventDefault();
      saveDraft();
    }
    // All other keys pass through normally for typing
  }

  function getBodyContent() {
    if (isHtml && htmlEditor) {
      return htmlEditor.innerHTML;
    }
    return bodyText;
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
        var body = getBodyContent();
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
          if (result.error && result.error.includes('Gmail API not configured')) {
            await handleOAuth2Required();
          } else {
            alert(`Erro: ${result.error}`);
          }
        }
      } else {
        info('[Mock] Email enviado com sucesso');
        close();
      }
    } catch (err) {
      logError('Erro ao enviar email', err);
      if (err.message && err.message.includes('Gmail API not configured')) {
        await handleOAuth2Required();
      } else {
        alert(`Erro: ${err.message}`);
      }
    } finally {
      sending = false;
    }
  }

  async function handleOAuth2Required() {
    var shouldAuth = confirm(
      'Gmail API nao esta configurado.\n\n' +
      'Voce precisa autenticar com sua conta Google para enviar emails.\n\n' +
      'Deseja autenticar agora? (Abrira o navegador)'
    );

    if (shouldAuth) {
      info('Iniciando autenticacao OAuth2...');
      try {
        await window.go.desktop.App.StartOAuth2Auth();
        info('Autenticacao concluida! Tente enviar novamente.');
        alert('Autenticacao concluida com sucesso!\n\nClique em Enviar novamente.');
      } catch (authErr) {
        logError('Erro na autenticacao', authErr);
        alert(`Erro na autenticacao: ${authErr.message}`);
      }
    }
  }

  async function saveDraft() {
    info('Salvando rascunho...');
    try {
      if (window.go?.desktop?.App) {
        var body = getBodyContent();
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

  function handleHtmlInput() {
    // Keep bodyHtml synced with editor content
    if (htmlEditor) {
      bodyHtml = htmlEditor.innerHTML;
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="overlay" on:click={close} role="button" tabindex="-1">
  <div class="compose-modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="compose-header">
      <h2>{getTitle()}</h2>
      <div class="header-actions">
        <button
          class="mode-toggle"
          class:active={isHtml}
          on:click={toggleMode}
          title={isHtml ? 'Mudar para texto puro' : 'Mudar para HTML'}
        >
          {isHtml ? 'HTML' : 'Texto'}
        </button>
        <button class="close-btn" on:click={close}>✕</button>
      </div>
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
        {#if isHtml}
          <div
            class="html-editor"
            bind:this={htmlEditor}
            contenteditable="true"
            on:input={handleHtmlInput}
            role="textbox"
            aria-multiline="true"
          ></div>
        {:else}
          <textarea
            bind:value={bodyText}
            placeholder="Escreva sua mensagem..."
            rows="12"
          ></textarea>
        {/if}
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

  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
  }

  .mode-toggle {
    padding: 4px 10px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: var(--text-muted);
    font-size: var(--font-xs);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .mode-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .mode-toggle.active {
    background: var(--accent-primary);
    border-color: var(--accent-primary);
    color: white;
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

  .html-editor {
    flex: 1;
    width: 100%;
    min-height: 200px;
    padding: var(--space-md);
    background: white;
    border: 1px solid var(--border-color);
    border-radius: var(--radius-sm);
    color: #333;
    font-size: var(--font-sm);
    font-family: Arial, sans-serif;
    line-height: 1.6;
    overflow-y: auto;
  }

  .html-editor:focus {
    outline: none;
    border-color: var(--accent-primary);
  }

  .html-editor:empty::before {
    content: 'Escreva sua mensagem...';
    color: #999;
  }

  /* Style links and images inside the HTML editor */
  .html-editor :global(a) {
    color: #1a73e8;
  }

  .html-editor :global(img) {
    max-width: 100%;
    height: auto;
  }

  .html-editor :global(table) {
    border-collapse: collapse;
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
