<script>
  import { showHelp } from '../stores/ui.js';

  const shortcuts = [
    { section: 'Navegacao', items: [
      { key: 'j / ↓', desc: 'Proximo email' },
      { key: 'k / ↑', desc: 'Email anterior' },
      { key: 'Enter', desc: 'Abrir email' },
      { key: 'Tab', desc: 'Alternar paineis' },
      { key: 'Esc', desc: 'Fechar modal/overlay' },
    ]},
    { section: 'Multi-Select', items: [
      { key: 'v', desc: 'Ativar modo de selecao' },
      { key: 'Space', desc: 'Selecionar/deselecionar atual' },
      { key: 'Ctrl+A', desc: 'Selecionar todos' },
      { key: 'Shift+Click', desc: 'Selecao em lote' },
      { key: 'Ctrl+Click', desc: 'Alternar selecao' },
    ]},
    { section: 'Acoes', items: [
      { key: 'e', desc: 'Arquivar email' },
      { key: 'x / #', desc: 'Deletar email' },
      { key: 's', desc: 'Marcar/desmarcar estrela' },
      { key: 'u', desc: 'Marcar como nao lido' },
    ]},
    { section: 'Composicao', items: [
      { key: 'c', desc: 'Novo email' },
      { key: 'r', desc: 'Responder (ou sync se nao selecionado)' },
      { key: 'R', desc: 'Responder a todos' },
      { key: 'f', desc: 'Encaminhar' },
      { key: 'Ctrl+Enter', desc: 'Enviar (no compose)' },
    ]},
    { section: 'Busca & IA', items: [
      { key: '/', desc: 'Busca fuzzy' },
      { key: 'a', desc: 'Assistente IA' },
      { key: 'A', desc: 'IA com contexto do email' },
    ]},
    { section: 'Sistema', items: [
      { key: 'T', desc: 'Abrir modo Terminal' },
      { key: 'D', desc: 'Toggle Debug panel' },
      { key: 'S', desc: 'Configuracoes' },
      { key: '?', desc: 'Esta ajuda' },
      { key: 'q', desc: 'Sair' },
    ]},
  ];

  function close() {
    showHelp.set(false);
  }

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      close();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="overlay" on:click={close} role="button" tabindex="-1" on:keydown={handleKeydown}>
  <div class="help-modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="help-header">
      <h2>Atalhos de Teclado</h2>
      <button class="close-btn" on:click={close}>✕</button>
    </div>

    <div class="help-content">
      {#each shortcuts as section}
        <div class="shortcut-section">
          <h3>{section.section}</h3>
          <div class="shortcut-list">
            {#each section.items as item}
              <div class="shortcut-item">
                <kbd>{item.key}</kbd>
                <span>{item.desc}</span>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>

    <div class="help-footer">
      <span>Pressione <kbd>?</kbd> ou <kbd>Esc</kbd> para fechar</span>
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
    z-index: 2000;
  }

  .help-modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--radius-lg);
    width: 90%;
    max-width: 700px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .help-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-md) var(--space-lg);
    border-bottom: 1px solid var(--border-color);
  }

  .help-header h2 {
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

  .help-content {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-lg);
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--space-lg);
  }

  .shortcut-section h3 {
    font-size: var(--font-sm);
    font-weight: 600;
    color: var(--accent-primary);
    margin: 0 0 var(--space-sm) 0;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .shortcut-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-xs);
  }

  .shortcut-item {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    font-size: var(--font-sm);
  }

  .shortcut-item kbd {
    min-width: 50px;
    padding: 2px 6px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-family: monospace;
    font-size: 11px;
    text-align: center;
  }

  .shortcut-item span {
    color: var(--text-secondary);
  }

  .help-footer {
    padding: var(--space-sm) var(--space-lg);
    border-top: 1px solid var(--border-color);
    text-align: center;
    font-size: var(--font-xs);
    color: var(--text-muted);
  }

  .help-footer kbd {
    padding: 1px 4px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    font-family: monospace;
  }
</style>
