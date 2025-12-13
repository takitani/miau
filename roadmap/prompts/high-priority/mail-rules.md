# Prompt: Mail Rules (Regras Automáticas)

> Inspirado no Mailspring/Outlook - Automatize ações baseadas em condições.

## Conceito

Criar regras que executam ações automaticamente quando emails chegam. Similar a filtros do Gmail, mas mais poderoso e local.

```
┌─ Regras ─────────────────────────────────────────────────────┐
│                                                              │
│ 📋 Minhas Regras                                             │
│                                                              │
│ ✅ 1. Newsletters → Feed                                     │
│    SE header "List-Unsubscribe" existe                       │
│    ENTÃO mover para "Feed"                                   │
│                                                              │
│ ✅ 2. Faturas → Paper Trail + Estrelar                       │
│    SE assunto contém "fatura" OU "invoice" OU "NF-e"        │
│    ENTÃO mover para "Paper Trail" E marcar com estrela       │
│                                                              │
│ ✅ 3. Spam de RH                                             │
│    SE de "@marketing.random.com"                            │
│    ENTÃO arquivar E marcar como lido                        │
│                                                              │
│ ⏸️ 4. Auto-resposta férias (pausada)                         │
│    SE qualquer email                                        │
│    ENTÃO enviar resposta automática                          │
│                                                              │
│ [+ Nova Regra]                                               │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Anatomia de uma Regra

```
REGRA = {
    nome: "Newsletters para Feed",
    condições: [
        { campo: "header", operador: "exists", valor: "List-Unsubscribe" },
        OU
        { campo: "from", operador: "contains", valor: "@newsletter" }
    ],
    ações: [
        { tipo: "move", destino: "Feed" },
        { tipo: "mark_read" }
    ],
    ativa: true,
    ordem: 1,
    parar_após: true  // Não processar mais regras
}
```

## Database Schema

```sql
-- Regras de email
CREATE TABLE mail_rules (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    name TEXT NOT NULL,
    is_active BOOLEAN DEFAULT 1,
    sort_order INTEGER DEFAULT 0,
    stop_processing BOOLEAN DEFAULT 1,  -- Parar após esta regra
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Condições da regra (AND entre elas, OR = criar outra regra)
CREATE TABLE mail_rule_conditions (
    id INTEGER PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES mail_rules(id) ON DELETE CASCADE,
    field TEXT NOT NULL,          -- from, to, subject, header, body, has_attachment
    operator TEXT NOT NULL,       -- contains, not_contains, equals, exists, regex
    value TEXT,                   -- Valor para comparar
    sort_order INTEGER DEFAULT 0
);

-- Ações da regra (executadas em ordem)
CREATE TABLE mail_rule_actions (
    id INTEGER PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES mail_rules(id) ON DELETE CASCADE,
    action_type TEXT NOT NULL,    -- move, copy, archive, delete, mark_read, mark_unread, star, label, forward, auto_reply
    action_value TEXT,            -- Valor dependendo do tipo (pasta destino, email forward, etc)
    sort_order INTEGER DEFAULT 0
);

-- Log de execução de regras
CREATE TABLE mail_rule_log (
    id INTEGER PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES mail_rules(id),
    email_id INTEGER NOT NULL REFERENCES emails(id),
    executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    actions_taken TEXT            -- JSON array de ações executadas
);

-- Índices
CREATE INDEX idx_rules_account ON mail_rules(account_id, is_active);
CREATE INDEX idx_rule_log ON mail_rule_log(rule_id, executed_at);
```

## Campos Disponíveis

```go
type RuleField string

const (
    FieldFrom          RuleField = "from"           // from_email
    FieldFromName      RuleField = "from_name"      // from_name
    FieldTo            RuleField = "to"             // to_addresses
    FieldCc            RuleField = "cc"             // cc_addresses
    FieldSubject       RuleField = "subject"        // subject
    FieldBody          RuleField = "body"           // body_text
    FieldHeader        RuleField = "header"         // raw_headers
    FieldHasAttachment RuleField = "has_attachment" // has_attachments
    FieldSize          RuleField = "size"           // size em bytes
    FieldDate          RuleField = "date"           // date
)
```

## Operadores

```go
type RuleOperator string

const (
    OpContains    RuleOperator = "contains"     // Contém texto
    OpNotContains RuleOperator = "not_contains" // Não contém
    OpEquals      RuleOperator = "equals"       // Igual exato
    OpNotEquals   RuleOperator = "not_equals"   // Diferente
    OpStartsWith  RuleOperator = "starts_with"  // Começa com
    OpEndsWith    RuleOperator = "ends_with"    // Termina com
    OpExists      RuleOperator = "exists"       // Header existe
    OpNotExists   RuleOperator = "not_exists"   // Header não existe
    OpRegex       RuleOperator = "regex"        // Expressão regular
    OpGreaterThan RuleOperator = "gt"           // Maior que (size)
    OpLessThan    RuleOperator = "lt"           // Menor que (size)
)
```

## Ações Disponíveis

```go
type RuleAction string

const (
    ActionMove       RuleAction = "move"        // Mover para pasta
    ActionCopy       RuleAction = "copy"        // Copiar para pasta
    ActionArchive    RuleAction = "archive"     // Arquivar
    ActionDelete     RuleAction = "delete"      // Mover para lixeira
    ActionMarkRead   RuleAction = "mark_read"   // Marcar como lido
    ActionMarkUnread RuleAction = "mark_unread" // Marcar como não lido
    ActionStar       RuleAction = "star"        // Adicionar estrela
    ActionUnstar     RuleAction = "unstar"      // Remover estrela
    ActionLabel      RuleAction = "label"       // Adicionar label
    ActionForward    RuleAction = "forward"     // Encaminhar para email
    ActionAutoReply  RuleAction = "auto_reply"  // Resposta automática
    ActionNotify     RuleAction = "notify"      // Notificação especial
)
```

## Service Implementation

```go
// internal/services/mail_rule.go

type MailRuleService struct {
    storage ports.MailRuleStorage
    email   ports.EmailService
    notify  ports.NotificationService
}

type Rule struct {
    ID             int64
    Name           string
    IsActive       bool
    SortOrder      int
    StopProcessing bool
    Conditions     []Condition
    Actions        []Action
}

type Condition struct {
    Field    RuleField
    Operator RuleOperator
    Value    string
}

type Action struct {
    Type  RuleAction
    Value string
}

// ProcessEmail aplica todas as regras a um email
func (s *MailRuleService) ProcessEmail(ctx context.Context, email *Email) error {
    rules, err := s.storage.GetActiveRules(ctx, email.AccountID)
    if err != nil {
        return err
    }

    for _, rule := range rules {
        if s.matchesRule(email, rule) {
            // Executar ações
            if err := s.executeActions(ctx, email, rule); err != nil {
                s.logger.Error("Rule action failed", "rule", rule.Name, "error", err)
                continue
            }

            // Log de execução
            s.storage.LogExecution(ctx, rule.ID, email.ID, rule.Actions)

            // Parar processamento se configurado
            if rule.StopProcessing {
                break
            }
        }
    }

    return nil
}

func (s *MailRuleService) matchesRule(email *Email, rule Rule) bool {
    // Todas as condições devem ser verdadeiras (AND)
    for _, cond := range rule.Conditions {
        if !s.matchesCondition(email, cond) {
            return false
        }
    }
    return true
}

func (s *MailRuleService) matchesCondition(email *Email, cond Condition) bool {
    // Obter valor do campo
    fieldValue := s.getFieldValue(email, cond.Field)

    switch cond.Operator {
    case OpContains:
        return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(cond.Value))

    case OpNotContains:
        return !strings.Contains(strings.ToLower(fieldValue), strings.ToLower(cond.Value))

    case OpEquals:
        return strings.EqualFold(fieldValue, cond.Value)

    case OpNotEquals:
        return !strings.EqualFold(fieldValue, cond.Value)

    case OpStartsWith:
        return strings.HasPrefix(strings.ToLower(fieldValue), strings.ToLower(cond.Value))

    case OpEndsWith:
        return strings.HasSuffix(strings.ToLower(fieldValue), strings.ToLower(cond.Value))

    case OpExists:
        // Para headers
        if cond.Field == FieldHeader {
            return strings.Contains(email.RawHeaders, cond.Value+":")
        }
        return fieldValue != ""

    case OpNotExists:
        if cond.Field == FieldHeader {
            return !strings.Contains(email.RawHeaders, cond.Value+":")
        }
        return fieldValue == ""

    case OpRegex:
        re, err := regexp.Compile(cond.Value)
        if err != nil {
            return false
        }
        return re.MatchString(fieldValue)

    case OpGreaterThan:
        size, _ := strconv.ParseInt(cond.Value, 10, 64)
        return email.Size > size

    case OpLessThan:
        size, _ := strconv.ParseInt(cond.Value, 10, 64)
        return email.Size < size
    }

    return false
}

func (s *MailRuleService) getFieldValue(email *Email, field RuleField) string {
    switch field {
    case FieldFrom:
        return email.FromEmail
    case FieldFromName:
        return email.FromName
    case FieldTo:
        return email.ToAddresses
    case FieldCc:
        return email.CcAddresses
    case FieldSubject:
        return email.Subject
    case FieldBody:
        return email.BodyText
    case FieldHeader:
        return email.RawHeaders
    case FieldHasAttachment:
        if email.HasAttachments {
            return "true"
        }
        return "false"
    case FieldSize:
        return strconv.FormatInt(email.Size, 10)
    case FieldDate:
        return email.Date.Format(time.RFC3339)
    }
    return ""
}

func (s *MailRuleService) executeActions(ctx context.Context, email *Email, rule Rule) error {
    for _, action := range rule.Actions {
        var err error

        switch action.Type {
        case ActionMove:
            err = s.email.MoveToFolder(ctx, email.ID, action.Value)

        case ActionCopy:
            err = s.email.CopyToFolder(ctx, email.ID, action.Value)

        case ActionArchive:
            err = s.email.Archive(ctx, email.ID)

        case ActionDelete:
            err = s.email.Delete(ctx, email.ID)

        case ActionMarkRead:
            err = s.email.MarkAsRead(ctx, email.ID)

        case ActionMarkUnread:
            err = s.email.MarkAsUnread(ctx, email.ID)

        case ActionStar:
            err = s.email.Star(ctx, email.ID)

        case ActionUnstar:
            err = s.email.Unstar(ctx, email.ID)

        case ActionLabel:
            err = s.email.AddLabel(ctx, email.ID, action.Value)

        case ActionForward:
            err = s.email.Forward(ctx, email.ID, action.Value)

        case ActionAutoReply:
            err = s.sendAutoReply(ctx, email, action.Value)

        case ActionNotify:
            err = s.notify.Send(ctx, "Rule: "+rule.Name, email.Subject)
        }

        if err != nil {
            return fmt.Errorf("action %s failed: %w", action.Type, err)
        }
    }

    return nil
}

// CreateRule cria nova regra
func (s *MailRuleService) CreateRule(ctx context.Context, accountID int64, rule *Rule) error {
    // Validar
    if err := s.validateRule(rule); err != nil {
        return err
    }

    return s.storage.Create(ctx, accountID, rule)
}

// TestRule testa uma regra contra emails existentes
func (s *MailRuleService) TestRule(ctx context.Context, accountID int64, rule *Rule, limit int) ([]Email, error) {
    // Buscar emails recentes
    emails, err := s.email.GetRecent(ctx, accountID, 100)
    if err != nil {
        return nil, err
    }

    // Filtrar os que matcham
    matches := make([]Email, 0)
    for _, email := range emails {
        if s.matchesRule(&email, *rule) {
            matches = append(matches, email)
            if len(matches) >= limit {
                break
            }
        }
    }

    return matches, nil
}
```

## Desktop UI

```svelte
<!-- RuleEditor.svelte -->
<script>
  import { CreateRule, UpdateRule, TestRule, GetFolders } from '../wailsjs/go/desktop/App';

  export let rule = null;  // null = nova regra
  export let onSave;
  export let onCancel;

  let name = rule?.name || '';
  let isActive = rule?.isActive ?? true;
  let stopProcessing = rule?.stopProcessing ?? true;
  let conditions = rule?.conditions || [{ field: 'from', operator: 'contains', value: '' }];
  let actions = rule?.actions || [{ type: 'move', value: '' }];

  let folders = [];
  let testResults = null;
  let testing = false;

  const fields = [
    { value: 'from', label: 'De (email)' },
    { value: 'from_name', label: 'De (nome)' },
    { value: 'to', label: 'Para' },
    { value: 'cc', label: 'Cc' },
    { value: 'subject', label: 'Assunto' },
    { value: 'body', label: 'Corpo' },
    { value: 'header', label: 'Header' },
    { value: 'has_attachment', label: 'Tem anexo' },
    { value: 'size', label: 'Tamanho' },
  ];

  const operators = [
    { value: 'contains', label: 'contém' },
    { value: 'not_contains', label: 'não contém' },
    { value: 'equals', label: 'é igual a' },
    { value: 'not_equals', label: 'é diferente de' },
    { value: 'starts_with', label: 'começa com' },
    { value: 'ends_with', label: 'termina com' },
    { value: 'exists', label: 'existe' },
    { value: 'not_exists', label: 'não existe' },
    { value: 'regex', label: 'regex' },
    { value: 'gt', label: 'maior que' },
    { value: 'lt', label: 'menor que' },
  ];

  const actionTypes = [
    { value: 'move', label: 'Mover para pasta' },
    { value: 'copy', label: 'Copiar para pasta' },
    { value: 'archive', label: 'Arquivar' },
    { value: 'delete', label: 'Mover para lixeira' },
    { value: 'mark_read', label: 'Marcar como lido' },
    { value: 'mark_unread', label: 'Marcar como não lido' },
    { value: 'star', label: 'Adicionar estrela' },
    { value: 'unstar', label: 'Remover estrela' },
    { value: 'label', label: 'Adicionar label' },
    { value: 'forward', label: 'Encaminhar para' },
    { value: 'auto_reply', label: 'Resposta automática' },
    { value: 'notify', label: 'Notificação especial' },
  ];

  onMount(async () => {
    folders = await GetFolders();
  });

  function addCondition() {
    conditions = [...conditions, { field: 'from', operator: 'contains', value: '' }];
  }

  function removeCondition(index) {
    conditions = conditions.filter((_, i) => i !== index);
  }

  function addAction() {
    actions = [...actions, { type: 'archive', value: '' }];
  }

  function removeAction(index) {
    actions = actions.filter((_, i) => i !== index);
  }

  async function testRule() {
    testing = true;
    testResults = null;

    const ruleData = { name, isActive, stopProcessing, conditions, actions };
    testResults = await TestRule(ruleData, 10);

    testing = false;
  }

  async function save() {
    const ruleData = { name, isActive, stopProcessing, conditions, actions };

    if (rule?.id) {
      await UpdateRule(rule.id, ruleData);
    } else {
      await CreateRule(ruleData);
    }

    onSave?.();
  }

  function needsValue(actionType) {
    return ['move', 'copy', 'label', 'forward', 'auto_reply'].includes(actionType);
  }

  function needsFolderSelect(actionType) {
    return ['move', 'copy'].includes(actionType);
  }
</script>

<div class="rule-editor">
  <h2>{rule ? 'Editar' : 'Nova'} Regra</h2>

  <div class="field">
    <label>Nome da regra</label>
    <input type="text" bind:value={name} placeholder="Ex: Newsletters para Feed" />
  </div>

  <!-- Condições -->
  <div class="section">
    <h3>Quando o email...</h3>

    {#each conditions as cond, i}
      <div class="condition-row">
        <select bind:value={cond.field}>
          {#each fields as f}
            <option value={f.value}>{f.label}</option>
          {/each}
        </select>

        <select bind:value={cond.operator}>
          {#each operators as op}
            <option value={op.value}>{op.label}</option>
          {/each}
        </select>

        {#if !['exists', 'not_exists'].includes(cond.operator)}
          <input type="text" bind:value={cond.value} placeholder="Valor" />
        {/if}

        {#if conditions.length > 1}
          <button class="remove" on:click={() => removeCondition(i)}>×</button>
        {/if}
      </div>

      {#if i < conditions.length - 1}
        <div class="logic-operator">E</div>
      {/if}
    {/each}

    <button class="add" on:click={addCondition}>+ Adicionar condição</button>
  </div>

  <!-- Ações -->
  <div class="section">
    <h3>Então...</h3>

    {#each actions as action, i}
      <div class="action-row">
        <select bind:value={action.type}>
          {#each actionTypes as at}
            <option value={at.value}>{at.label}</option>
          {/each}
        </select>

        {#if needsFolderSelect(action.type)}
          <select bind:value={action.value}>
            {#each folders as folder}
              <option value={folder.name}>{folder.name}</option>
            {/each}
          </select>
        {:else if needsValue(action.type)}
          <input type="text" bind:value={action.value} placeholder="Valor" />
        {/if}

        {#if actions.length > 1}
          <button class="remove" on:click={() => removeAction(i)}>×</button>
        {/if}
      </div>
    {/each}

    <button class="add" on:click={addAction}>+ Adicionar ação</button>
  </div>

  <!-- Opções -->
  <div class="options">
    <label>
      <input type="checkbox" bind:checked={isActive} />
      Regra ativa
    </label>

    <label>
      <input type="checkbox" bind:checked={stopProcessing} />
      Parar de processar outras regras após esta
    </label>
  </div>

  <!-- Teste -->
  <div class="test-section">
    <button on:click={testRule} disabled={testing}>
      {testing ? 'Testando...' : 'Testar regra'}
    </button>

    {#if testResults}
      <div class="test-results">
        <h4>{testResults.length} email(s) correspondem:</h4>
        {#each testResults as email}
          <div class="test-email">
            <span class="from">{email.fromEmail}</span>
            <span class="subject">{email.subject}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Botões -->
  <div class="buttons">
    <button class="secondary" on:click={onCancel}>Cancelar</button>
    <button class="primary" on:click={save} disabled={!name}>Salvar</button>
  </div>
</div>

<style>
  .rule-editor {
    padding: var(--space-lg);
    max-width: 700px;
  }

  .section {
    margin: var(--space-lg) 0;
    padding: var(--space-md);
    background: var(--bg-secondary);
    border-radius: var(--radius-md);
  }

  .section h3 {
    margin: 0 0 var(--space-md);
    color: var(--text-secondary);
  }

  .condition-row,
  .action-row {
    display: flex;
    gap: var(--space-sm);
    margin-bottom: var(--space-sm);
    align-items: center;
  }

  select, input[type="text"] {
    padding: var(--space-sm);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-sm);
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  select {
    min-width: 150px;
  }

  input[type="text"] {
    flex: 1;
  }

  .logic-operator {
    text-align: center;
    color: var(--text-tertiary);
    font-size: 0.85em;
    margin: var(--space-xs) 0;
  }

  .add {
    margin-top: var(--space-sm);
    padding: var(--space-xs) var(--space-sm);
    background: transparent;
    border: 1px dashed var(--border-subtle);
    color: var(--text-secondary);
    cursor: pointer;
  }

  .add:hover {
    border-color: var(--accent-primary);
    color: var(--accent-primary);
  }

  .remove {
    padding: var(--space-xs) var(--space-sm);
    background: transparent;
    border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    font-size: 1.2em;
  }

  .remove:hover {
    color: var(--text-error);
  }

  .options {
    display: flex;
    flex-direction: column;
    gap: var(--space-sm);
  }

  .options label {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    cursor: pointer;
  }

  .test-section {
    margin: var(--space-lg) 0;
    padding: var(--space-md);
    background: var(--bg-tertiary);
    border-radius: var(--radius-md);
  }

  .test-results {
    margin-top: var(--space-md);
  }

  .test-email {
    padding: var(--space-xs);
    font-size: 0.9em;
  }

  .test-email .from {
    color: var(--text-secondary);
    margin-right: var(--space-sm);
  }

  .buttons {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-sm);
    margin-top: var(--space-lg);
  }

  .buttons button {
    padding: var(--space-sm) var(--space-lg);
  }

  .primary {
    background: var(--accent-primary);
    color: white;
  }

  .secondary {
    background: transparent;
    border: 1px solid var(--border-subtle);
  }
</style>
```

## Regras Pré-definidas (Sugeridas)

```go
var suggestedRules = []Rule{
    {
        Name: "Newsletters para Feed",
        Conditions: []Condition{
            {Field: FieldHeader, Operator: OpExists, Value: "List-Unsubscribe"},
        },
        Actions: []Action{
            {Type: ActionMove, Value: "Feed"},
            {Type: ActionMarkRead},
        },
    },
    {
        Name: "Faturas e Recibos",
        Conditions: []Condition{
            {Field: FieldSubject, Operator: OpRegex, Value: "(?i)(fatura|invoice|nota fiscal|NF-e|recibo|payment)"},
        },
        Actions: []Action{
            {Type: ActionMove, Value: "Paper Trail"},
            {Type: ActionStar},
        },
    },
    {
        Name: "GitHub Notifications",
        Conditions: []Condition{
            {Field: FieldFrom, Operator: OpContains, Value: "notifications@github.com"},
        },
        Actions: []Action{
            {Type: ActionLabel, Value: "GitHub"},
        },
    },
    {
        Name: "Emails grandes (>10MB)",
        Conditions: []Condition{
            {Field: FieldSize, Operator: OpGreaterThan, Value: "10485760"},
        },
        Actions: []Action{
            {Type: ActionLabel, Value: "Anexos Grandes"},
        },
    },
}
```

## TUI Implementation

```go
// Tecla 'R' (Shift+R) para regras
case "R":
    return m.showRulesManager()

// Rules Manager
type RulesModel struct {
    rules    []Rule
    selected int
    editing  *Rule
}

func (m RulesModel) View() string {
    var b strings.Builder

    b.WriteString("📋 Regras de Email\n")
    b.WriteString("─────────────────────────────────────\n\n")

    for i, rule := range m.rules {
        cursor := "  "
        if i == m.selected {
            cursor = "▸ "
        }

        status := "✅"
        if !rule.IsActive {
            status = "⏸️"
        }

        b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, status, rule.Name))

        // Mostrar resumo
        if len(rule.Conditions) > 0 {
            b.WriteString(fmt.Sprintf("     SE %s %s %s\n",
                rule.Conditions[0].Field,
                rule.Conditions[0].Operator,
                rule.Conditions[0].Value,
            ))
        }
        if len(rule.Actions) > 0 {
            b.WriteString(fmt.Sprintf("     ENTÃO %s\n", rule.Actions[0].Type))
        }
        b.WriteString("\n")
    }

    b.WriteString("\n[Enter] Editar  [n] Nova  [d] Deletar  [Space] Ativar/Desativar  [q] Voltar")

    return b.String()
}
```

## Integração com Sync

```go
// No sync service, após salvar novo email
func (s *SyncService) onNewEmail(ctx context.Context, email *Email) error {
    // ... salvar email ...

    // Processar regras
    if err := s.ruleService.ProcessEmail(ctx, email); err != nil {
        s.logger.Warn("Rule processing failed", "error", err)
        // Não falhar o sync por causa de regras
    }

    return nil
}
```

## Critérios de Aceitação

- [ ] Regras podem ser criadas/editadas/deletadas
- [ ] Todas as condições devem ser verdadeiras (AND)
- [ ] Múltiplas ações executam em ordem
- [ ] Teste de regra funciona
- [ ] Regras aplicadas a novos emails
- [ ] Log de execução disponível
- [ ] Regras sugeridas disponíveis
- [ ] Performance: < 10ms por email
- [ ] UI Desktop e TUI implementadas

---

*Inspirado em: Mailspring, Outlook, Gmail Filters*
