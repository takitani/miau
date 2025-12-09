# SonarQube Issues - miau

> Relatório gerado em: 2025-12-06
> Branch: master
> Quality Gate: Passed

## Resumo

| Categoria | Issues | Esforço |
|-----------|--------|---------|
| Maintainability | 349 | 7d 3h |
| Reliability | 25 | - |
| Security | 0 | - |

### Clean Code Attributes

| Atributo | Issues |
|----------|--------|
| Intentionality | 193 |
| Adaptability | 155 |
| Consistency | 1 |
| Responsibility | 0 |

---

## Issues por Prioridade

### CRITICAL - Corrigir Imediatamente

#### 1. "Unexpected var, use let or const instead" (JavaScript)
- **Severity**: Critical
- **Effort**: 5min each
- **Files afetados** (stores JavaScript):
  - `cmd/miau-desktop/frontend/src/lib/stores/calendar.js`
  - `cmd/miau-desktop/frontend/src/lib/stores/analytics.js`
  - `cmd/miau-desktop/frontend/src/lib/stores/contacts.js`
  - `cmd/miau-desktop/frontend/src/lib/stores/tasks.js`
  - `cmd/miau-desktop/frontend/src/lib/stores/ui.js`
  - E outros...

**Problema**: Uso de `var` em JavaScript moderno. Deve usar `const` ou `let`.

**Nota importante**: O CLAUDE.md especifica "use var nas declaracoes de variaveis" mas isso é para **Go**, não JavaScript! Em JS, sempre usar `const` (preferido) ou `let`.

**Correção**:
```javascript
// Errado
var emails = writable([]);

// Correto
const emails = writable([]);
```

---

### HIGH - Corrigir em Breve

#### 2. "Exporting mutable 'var' binding, use 'const' instead"
- **Severity**: High (Maintainability) + Medium (Reliability)
- **Effort**: 5min each
- **Files**: Mesmos stores acima

**Problema**: Exportar variáveis mutáveis (`var`) pode causar bugs difíceis de rastrear.

**Correção**: Usar `const` para stores Svelte exportados.

---

### MEDIUM - Melhorias de Código

#### 3. "Prefer using an optional chain expression instead"
- **Severity**: Medium
- **Effort**: 5min each
- **Files**: Vários componentes Svelte e stores

**Problema**: Usar `&&` para acessar propriedades aninhadas quando `?.` é mais limpo.

**Correção**:
```javascript
// Errado
if (data && data.user && data.user.name) { ... }

// Correto
if (data?.user?.name) { ... }
```

---

### LOW - Nice to Have

#### 4. "Remove this unused import"
- **Severity**: Low
- **Effort**: 1min each
- **Files**:
  - `analytics.js` - import `get` não usado
  - `calendar.js` - import `GetCalendarEvents` não usado

**Correção**: Remover imports não utilizados.

---

## Plano de Ação

### Fase 1: Quick Wins (1-2 horas)
- [ ] Remover imports não utilizados
- [ ] Substituir `var` por `const` nos stores principais

### Fase 2: Refactoring (4-6 horas)
- [ ] Aplicar optional chaining onde aplicável
- [ ] Revisar todos os stores para consistência

### Fase 3: Verificação
- [ ] Rodar `npm run lint` no frontend
- [ ] Re-scan no SonarQube
- [ ] Verificar que o app ainda funciona

---

## Arquivos Mais Afetados

| Arquivo | Issues | Prioridade |
|---------|--------|------------|
| `stores/calendar.js` | ~20+ | Alta |
| `stores/analytics.js` | ~10+ | Alta |
| `stores/contacts.js` | ~10+ | Alta |
| `stores/tasks.js` | ~10+ | Alta |
| `stores/ui.js` | ~5+ | Média |

---

## Nota sobre var vs const/let

**Em Go**: Usar `var` é idiomático e correto.
```go
var emails []Email  // Correto em Go
```

**Em JavaScript**: Usar `const` (preferido) ou `let`. Nunca `var`.
```javascript
const emails = writable([]);  // Correto em JS
let counter = 0;              // Para valores que mudam
```

---

## Links Úteis

- [SonarQube Dashboard](http://localhost:9000/dashboard?id=miau)
- [Issues Maintainability](http://localhost:9000/project/issues?issueStatuses=OPEN%2CCONFIRMED&impactSoftwareQualities=MAINTAINABILITY&branch=master&id=miau)
- [Issues Reliability](http://localhost:9000/project/issues?issueStatuses=OPEN%2CCONFIRMED&impactSoftwareQualities=RELIABILITY&branch=master&id=miau)
