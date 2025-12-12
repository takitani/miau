<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { AddAccount, StartOAuth2AuthForNewAccount, GetKnownImapHost, OpenURL } from '../../../bindings/github.com/opik/miau/internal/desktop/app.js';

  const dispatch = createEventDispatcher();

  // Open external URL
  function openLink(url) {
    OpenURL(url);
  }

  export let isOpen = false;

  // Wizard steps
  const STEP_EMAIL = 0;
  const STEP_AUTH_TYPE = 1;
  const STEP_IMAP = 2;
  const STEP_PASSWORD = 3;
  const STEP_OAUTH2 = 4;
  const STEP_OAUTH2_AUTH = 5;
  const STEP_CONFIRM = 6;

  let step = STEP_EMAIL;
  let isLoading = false;
  let error = '';

  // Form data
  let email = '';
  let name = '';
  let authType = 'oauth2'; // 'oauth2' or 'password'
  let imapHost = 'imap.gmail.com';
  let imapPort = 993;
  let smtpHost = 'smtp.gmail.com';
  let smtpPort = 587;
  let password = '';
  let clientId = '';
  let clientSecret = '';
  let sendMethod = 'gmail_api';
  let isGoogle = true;

  // Reset form
  function resetForm() {
    step = STEP_EMAIL;
    error = '';
    email = '';
    name = '';
    authType = 'oauth2';
    imapHost = 'imap.gmail.com';
    imapPort = 993;
    smtpHost = 'smtp.gmail.com';
    smtpPort = 587;
    password = '';
    clientId = '';
    clientSecret = '';
    sendMethod = 'gmail_api';
    isGoogle = true;
    isLoading = false;
  }

  // Close modal
  function close() {
    resetForm();
    dispatch('close');
  }

  // Go to previous step
  function prevStep() {
    error = '';
    if (step === STEP_AUTH_TYPE) step = STEP_EMAIL;
    else if (step === STEP_IMAP) step = STEP_AUTH_TYPE;
    else if (step === STEP_PASSWORD) step = STEP_IMAP;
    else if (step === STEP_OAUTH2) step = STEP_IMAP;
    else if (step === STEP_OAUTH2_AUTH) step = STEP_OAUTH2;
    else if (step === STEP_CONFIRM) {
      if (authType === 'oauth2') step = STEP_OAUTH2_AUTH;
      else step = STEP_PASSWORD;
    }
  }

  // Go to next step
  async function nextStep() {
    error = '';

    if (step === STEP_EMAIL) {
      if (!email || !email.includes('@')) {
        error = 'Email inválido';
        return;
      }
      // Auto-detect host
      try {
        const hostInfo = await GetKnownImapHost(email);
        imapHost = hostInfo.imapHost || 'imap.gmail.com';
        imapPort = hostInfo.imapPort || 993;
        smtpHost = hostInfo.smtpHost || 'smtp.gmail.com';
        smtpPort = hostInfo.smtpPort || 587;
        isGoogle = hostInfo.isGoogle || false;
        sendMethod = hostInfo.sendMethod || 'smtp';

        // For personal Gmail, suggest App Password (much simpler)
        // OAuth2 only for Google Workspace with custom domain
        const isPersonalGmail = email.endsWith('@gmail.com') || email.endsWith('@googlemail.com');
        if (isPersonalGmail) {
          authType = 'password';
          sendMethod = 'smtp';
        } else if (isGoogle) {
          // Google Workspace (custom domain) - can use OAuth2
          authType = 'oauth2';
        } else {
          authType = 'password';
        }
      } catch (e) {
        console.error('Failed to get host info:', e);
      }
      if (!name) {
        name = email.split('@')[0];
      }
      step = STEP_AUTH_TYPE;
    }
    else if (step === STEP_AUTH_TYPE) {
      step = STEP_IMAP;
    }
    else if (step === STEP_IMAP) {
      if (!imapHost) {
        error = 'Host IMAP obrigatório';
        return;
      }
      if (imapPort <= 0) imapPort = 993;
      if (authType === 'oauth2') {
        step = STEP_OAUTH2;
      } else {
        step = STEP_PASSWORD;
      }
    }
    else if (step === STEP_PASSWORD) {
      if (!password) {
        error = 'Senha obrigatória';
        return;
      }
      step = STEP_CONFIRM;
    }
    else if (step === STEP_OAUTH2) {
      if (!clientId) {
        error = 'Client ID obrigatório';
        return;
      }
      if (!clientSecret) {
        error = 'Client Secret obrigatório';
        return;
      }
      step = STEP_OAUTH2_AUTH;
    }
    else if (step === STEP_OAUTH2_AUTH) {
      // Start OAuth2 flow
      isLoading = true;
      try {
        await StartOAuth2AuthForNewAccount(email, clientId, clientSecret);
        step = STEP_CONFIRM;
      } catch (e) {
        error = `Falha na autenticação: ${e}`;
      } finally {
        isLoading = false;
      }
    }
    else if (step === STEP_CONFIRM) {
      // Save account
      isLoading = true;
      try {
        await AddAccount({
          email,
          name,
          authType,
          password: authType === 'password' ? password : '',
          clientId: authType === 'oauth2' ? clientId : '',
          clientSecret: authType === 'oauth2' ? clientSecret : '',
          imapHost,
          imapPort,
          smtpHost,
          smtpPort,
          sendMethod
        });
        dispatch('added', { email });
        close();
      } catch (e) {
        error = `Falha ao salvar: ${e}`;
      } finally {
        isLoading = false;
      }
    }
  }

  // Handle keyboard
  function handleKeydown(e) {
    if (!isOpen) return;
    if (e.key === 'Escape') {
      close();
    } else if (e.key === 'Enter' && !isLoading) {
      nextStep();
    }
  }

  onMount(() => {
    document.addEventListener('keydown', handleKeydown);
    return () => document.removeEventListener('keydown', handleKeydown);
  });

  $: if (isOpen) {
    resetForm();
  }
</script>

{#if isOpen}
  <div class="modal-overlay" on:click={close}>
    <div class="modal" on:click|stopPropagation>
      <div class="modal-header">
        <h2>Adicionar Conta</h2>
        <button class="close-btn" on:click={close}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <div class="modal-content">
        <!-- Step indicator -->
        <div class="step-indicator">
          <span class="step-number">{step + 1}</span>
          <span class="step-total">/ 7</span>
        </div>

        <!-- Step: Email -->
        {#if step === STEP_EMAIL}
          <div class="step-content">
            <h3>Qual seu email?</h3>
            <input
              type="email"
              bind:value={email}
              placeholder="seu@email.com"
              class="input"
              autofocus
            />
          </div>
        {/if}

        <!-- Step: Auth Type -->
        {#if step === STEP_AUTH_TYPE}
          <div class="step-content">
            <h3>Tipo de autenticação</h3>
            <div class="auth-options">
              <button
                class="auth-option"
                class:selected={authType === 'oauth2'}
                on:click={() => authType = 'oauth2'}
              >
                <div class="option-icon">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                  </svg>
                </div>
                <div class="option-info">
                  <span class="option-title">OAuth2 (Google)</span>
                  <span class="option-desc">Mais seguro, abre navegador para login</span>
                </div>
                {#if authType === 'oauth2'}
                  <svg class="check" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 6L9 17l-5-5"/>
                  </svg>
                {/if}
              </button>
              <button
                class="auth-option"
                class:selected={authType === 'password'}
                on:click={() => authType = 'password'}
              >
                <div class="option-icon">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                    <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                  </svg>
                </div>
                <div class="option-info">
                  <span class="option-title">Senha / App Password</span>
                  <span class="option-desc">Use App Password se tiver 2FA</span>
                </div>
                {#if authType === 'password'}
                  <svg class="check" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 6L9 17l-5-5"/>
                  </svg>
                {/if}
              </button>
            </div>
          </div>
        {/if}

        <!-- Step: IMAP -->
        {#if step === STEP_IMAP}
          <div class="step-content">
            <h3>Servidor IMAP</h3>
            <div class="form-group">
              <label>Host</label>
              <input
                type="text"
                bind:value={imapHost}
                placeholder="imap.gmail.com"
                class="input"
              />
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>Porta</label>
                <input
                  type="number"
                  bind:value={imapPort}
                  placeholder="993"
                  class="input"
                />
              </div>
            </div>
          </div>
        {/if}

        <!-- Step: Password -->
        {#if step === STEP_PASSWORD}
          <div class="step-content">
            <h3>Senha</h3>
            {#if isGoogle}
              <div class="app-password-instructions">
                <p><strong>Para Gmail, use App Password:</strong></p>
                <ol>
                  <li>Ative <button class="link-btn" on:click={() => openLink('https://myaccount.google.com/security')}>Verificação em 2 etapas</button> (se não tiver)</li>
                  <li>Acesse <button class="link-btn" on:click={() => openLink('https://myaccount.google.com/apppasswords')}>App Passwords</button></li>
                  <li>Crie uma senha para "Mail" ou "Other (miau)"</li>
                  <li>Cole a senha de 16 caracteres abaixo</li>
                </ol>
              </div>
            {:else}
              <p class="hint">
                Digite sua senha do email.
              </p>
            {/if}
            <input
              type="password"
              bind:value={password}
              placeholder={isGoogle ? "xxxx xxxx xxxx xxxx" : "Senha"}
              class="input"
              autofocus
            />
          </div>
        {/if}

        <!-- Step: OAuth2 Credentials -->
        {#if step === STEP_OAUTH2}
          <div class="step-content">
            <h3>Credenciais OAuth2</h3>
            <div class="oauth-instructions">
              <p>Para obter as credenciais:</p>
              <ol>
                <li>Acesse <button class="link-btn" on:click={() => openLink('https://console.cloud.google.com')}>console.cloud.google.com</button></li>
                <li>Crie um projeto (ou use existente)</li>
                <li>APIs & Services → OAuth consent screen</li>
                <li>APIs & Services → Credentials → Create OAuth client ID</li>
                <li>Tipo: Desktop app</li>
              </ol>
            </div>
            <div class="form-group">
              <label>Client ID</label>
              <input
                type="text"
                bind:value={clientId}
                placeholder="xxxxx.apps.googleusercontent.com"
                class="input"
              />
            </div>
            <div class="form-group">
              <label>Client Secret</label>
              <input
                type="password"
                bind:value={clientSecret}
                placeholder="GOCSPX-xxxxx"
                class="input"
              />
            </div>
          </div>
        {/if}

        <!-- Step: OAuth2 Auth -->
        {#if step === STEP_OAUTH2_AUTH}
          <div class="step-content center">
            {#if isLoading}
              <div class="loading-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
                </svg>
              </div>
              <h3>Autenticando...</h3>
              <p>O navegador deve abrir automaticamente.<br/>Faça login e autorize o acesso.</p>
            {:else}
              <div class="auth-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
                </svg>
              </div>
              <h3>Pronto para autenticar</h3>
              <p>Clique em Próximo para abrir o navegador<br/>e fazer login na sua conta Google.</p>
            {/if}
          </div>
        {/if}

        <!-- Step: Confirm -->
        {#if step === STEP_CONFIRM}
          <div class="step-content">
            <h3>Confirmar</h3>
            <div class="confirm-details">
              <div class="detail-row">
                <span class="detail-label">Email:</span>
                <span class="detail-value">{email}</span>
              </div>
              <div class="detail-row">
                <span class="detail-label">Nome:</span>
                <span class="detail-value">{name}</span>
              </div>
              <div class="detail-row">
                <span class="detail-label">Servidor:</span>
                <span class="detail-value">{imapHost}:{imapPort}</span>
              </div>
              <div class="detail-row">
                <span class="detail-label">Auth:</span>
                <span class="detail-value">{authType === 'oauth2' ? 'OAuth2' : 'Senha'}</span>
              </div>
              {#if authType === 'oauth2'}
                <div class="detail-row success">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 6L9 17l-5-5"/>
                  </svg>
                  <span>Token salvo</span>
                </div>
              {/if}
            </div>
          </div>
        {/if}

        <!-- Error -->
        {#if error}
          <div class="error">{error}</div>
        {/if}
      </div>

      <div class="modal-footer">
        {#if step > STEP_EMAIL}
          <button class="btn secondary" on:click={prevStep} disabled={isLoading}>
            Voltar
          </button>
        {:else}
          <button class="btn secondary" on:click={close}>
            Cancelar
          </button>
        {/if}
        <button class="btn primary" on:click={nextStep} disabled={isLoading}>
          {#if isLoading}
            Aguarde...
          {:else if step === STEP_CONFIRM}
            Salvar
          {:else if step === STEP_OAUTH2_AUTH}
            Autenticar
          {:else}
            Próximo
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 150ms ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 480px;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow-lg);
    animation: slideUp 200ms ease;
  }

  @keyframes slideUp {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .modal-header h2 {
    margin: 0;
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--text-primary);
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .modal-content {
    flex: 1;
    padding: var(--space-lg);
    overflow-y: auto;
  }

  .step-indicator {
    display: flex;
    align-items: baseline;
    gap: 2px;
    margin-bottom: var(--space-md);
    color: var(--text-muted);
    font-size: var(--font-sm);
  }

  .step-number {
    font-size: var(--font-lg);
    font-weight: 600;
    color: var(--accent-primary);
  }

  .step-content {
    animation: fadeIn 200ms ease;
  }

  .step-content h3 {
    margin: 0 0 var(--space-md) 0;
    font-size: var(--font-md);
    font-weight: 500;
    color: var(--text-primary);
  }

  .step-content.center {
    text-align: center;
    padding: var(--space-lg) 0;
  }

  .step-content.center p {
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .input {
    width: 100%;
    padding: var(--space-sm) var(--space-md);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    background: var(--bg-primary);
    color: var(--text-primary);
    font-size: var(--font-md);
    transition: all var(--transition-fast);
  }

  .input:focus {
    outline: none;
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 3px rgba(255, 107, 107, 0.1);
  }

  .input::placeholder {
    color: var(--text-muted);
  }

  .form-group {
    margin-bottom: var(--space-md);
  }

  .form-group label {
    display: block;
    margin-bottom: var(--space-xs);
    font-size: var(--font-sm);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .form-row {
    display: flex;
    gap: var(--space-md);
  }

  .form-row .form-group {
    flex: 1;
  }

  .hint {
    margin-bottom: var(--space-md);
    font-size: var(--font-sm);
    color: var(--text-muted);
    line-height: 1.5;
  }

  .app-password-instructions {
    margin-bottom: var(--space-md);
    padding: var(--space-md);
    background: var(--bg-primary);
    border-radius: var(--radius-md);
    border-left: 3px solid var(--accent-primary);
  }

  .app-password-instructions p {
    margin: 0 0 var(--space-sm) 0;
    font-size: var(--font-sm);
    color: var(--text-primary);
  }

  .app-password-instructions ol {
    margin: 0;
    padding-left: var(--space-lg);
    font-size: var(--font-sm);
    color: var(--text-secondary);
  }

  .app-password-instructions li {
    margin-bottom: var(--space-xs);
  }

  .app-password-instructions a,
  .link-btn {
    color: var(--accent-primary);
    text-decoration: none;
    background: none;
    border: none;
    padding: 0;
    font: inherit;
    cursor: pointer;
  }

  .app-password-instructions a:hover,
  .link-btn:hover {
    text-decoration: underline;
  }

  .auth-options {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }

  .auth-option {
    display: flex;
    align-items: center;
    gap: var(--space-md);
    padding: var(--space-md);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-md);
    background: var(--bg-primary);
    cursor: pointer;
    transition: all var(--transition-fast);
    text-align: left;
  }

  .auth-option:hover {
    border-color: var(--accent-primary);
  }

  .auth-option.selected {
    border-color: var(--accent-primary);
    background: rgba(255, 107, 107, 0.05);
  }

  .option-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: var(--radius-md);
    background: var(--bg-secondary);
    color: var(--text-secondary);
  }

  .auth-option.selected .option-icon {
    background: var(--accent-primary);
    color: white;
  }

  .option-info {
    flex: 1;
  }

  .option-title {
    display: block;
    font-weight: 500;
    color: var(--text-primary);
  }

  .option-desc {
    display: block;
    font-size: var(--font-xs);
    color: var(--text-muted);
    margin-top: 2px;
  }

  .auth-option .check {
    color: var(--accent-primary);
  }

  .oauth-instructions {
    margin-bottom: var(--space-md);
    padding: var(--space-md);
    background: var(--bg-primary);
    border-radius: var(--radius-md);
    font-size: var(--font-sm);
    color: var(--text-secondary);
  }

  .oauth-instructions p {
    margin: 0 0 var(--space-sm) 0;
    font-weight: 500;
  }

  .oauth-instructions ol {
    margin: 0;
    padding-left: var(--space-lg);
  }

  .oauth-instructions li {
    margin-bottom: var(--space-xs);
  }

  .oauth-instructions a {
    color: var(--accent-primary);
    text-decoration: none;
  }

  .oauth-instructions a:hover {
    text-decoration: underline;
  }

  .loading-icon, .auth-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 80px;
    height: 80px;
    margin: 0 auto var(--space-md);
    border-radius: 50%;
    background: var(--bg-primary);
    color: var(--accent-primary);
  }

  .loading-icon {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .confirm-details {
    padding: var(--space-md);
    background: var(--bg-primary);
    border-radius: var(--radius-md);
  }

  .detail-row {
    display: flex;
    justify-content: space-between;
    padding: var(--space-xs) 0;
    border-bottom: 1px solid var(--border-color);
  }

  .detail-row:last-child {
    border-bottom: none;
  }

  .detail-row.success {
    justify-content: flex-start;
    gap: var(--space-xs);
    color: var(--success-color, #73D216);
    padding-top: var(--space-sm);
  }

  .detail-label {
    color: var(--text-muted);
    font-size: var(--font-sm);
  }

  .detail-value {
    color: var(--text-primary);
    font-weight: 500;
  }

  .error {
    margin-top: var(--space-md);
    padding: var(--space-sm) var(--space-md);
    background: rgba(255, 107, 107, 0.1);
    border: 1px solid var(--accent-primary);
    border-radius: var(--radius-md);
    color: var(--accent-primary);
    font-size: var(--font-sm);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-sm);
    padding: var(--space-md) var(--space-lg);
    border-top: 1px solid var(--border-color);
  }

  .btn {
    padding: var(--space-sm) var(--space-lg);
    border: none;
    border-radius: var(--radius-md);
    font-size: var(--font-sm);
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn.primary {
    background: var(--accent-primary);
    color: white;
  }

  .btn.primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .btn.secondary {
    background: var(--bg-tertiary);
    color: var(--text-primary);
  }

  .btn.secondary:hover:not(:disabled) {
    background: var(--bg-hover);
  }
</style>
