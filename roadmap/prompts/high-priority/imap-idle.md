# Prompt: IMAP IDLE (Push Notifications)

> Use este prompt com Claude Code para implementar IMAP IDLE para notificações em tempo real.

## Contexto

Atualmente o miau usa polling para verificar novos emails. IMAP IDLE permite receber notificações push quando novos emails chegam.

## Objetivo

Implementar IMAP IDLE para:
1. Receber notificações de novos emails instantaneamente
2. Reduzir uso de banda/bateria
3. Melhorar UX com sync em tempo real

## Arquivos Relevantes

```
internal/imap/client.go          # Cliente IMAP atual
internal/services/sync.go        # Service de sync
internal/ports/events.go         # Event bus
go.mod                           # go-imap/v2 já suporta IDLE
```

## Como IMAP IDLE Funciona

```
Cliente                     Servidor
   |                           |
   |---- IDLE ---------------->|
   |                           |  (espera até 29 min)
   |<--- * EXISTS 15 ---------|  (novo email!)
   |                           |
   |---- DONE ---------------->|
   |<--- OK IDLE completed ----|
   |                           |
   |---- FETCH ... ----------->|  (buscar novo email)
```

## Tasks

### 1. IMAP Client - IDLE Support

Adicionar ao `internal/imap/client.go`:

```go
import "github.com/emersion/go-imap/v2/imapclient"

// IDLEConfig configura o comportamento do IDLE
type IDLEConfig struct {
    Timeout     time.Duration // Max 29 min (RFC)
    RetryDelay  time.Duration // Delay entre retries
    MaxRetries  int           // Max retries antes de desistir
}

// StartIDLE inicia o loop de IDLE
func (c *Client) StartIDLE(ctx context.Context, mailbox string, onUpdate func(IDLEUpdate)) error {
    // 1. Selecionar mailbox
    if err := c.SelectMailbox(mailbox); err != nil {
        return err
    }

    // 2. Loop de IDLE
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := c.doIDLE(ctx, onUpdate); err != nil {
                if !c.shouldRetry(err) {
                    return err
                }
                time.Sleep(c.config.RetryDelay)
            }
        }
    }
}

func (c *Client) doIDLE(ctx context.Context, onUpdate func(IDLEUpdate)) error {
    idleCmd := c.client.Idle()

    // Timeout de 28 minutos (RFC recomenda < 29)
    timer := time.NewTimer(28 * time.Minute)
    defer timer.Stop()

    for {
        select {
        case <-ctx.Done():
            idleCmd.Close()
            return ctx.Err()

        case <-timer.C:
            // Refresh IDLE antes do timeout
            idleCmd.Close()
            return nil

        case update := <-idleCmd.Updates():
            switch u := update.(type) {
            case *imapclient.UnilateralDataMailbox:
                if u.NumMessages != nil {
                    onUpdate(IDLEUpdate{
                        Type:     UpdateNewMail,
                        Mailbox:  c.currentMailbox,
                        NewCount: *u.NumMessages,
                    })
                }
            case *imapclient.UnilateralDataExpunge:
                onUpdate(IDLEUpdate{
                    Type:    UpdateExpunge,
                    Mailbox: c.currentMailbox,
                    SeqNum:  u.SeqNum,
                })
            }
        }
    }
}

type IDLEUpdate struct {
    Type     IDLEUpdateType
    Mailbox  string
    NewCount uint32
    SeqNum   uint32
}

type IDLEUpdateType int

const (
    UpdateNewMail IDLEUpdateType = iota
    UpdateExpunge
    UpdateFlags
)
```

### 2. Sync Service - IDLE Integration

Adicionar ao `internal/services/sync.go`:

```go
// StartRealtimeSync inicia sync em tempo real via IDLE
func (s *SyncService) StartRealtimeSync(ctx context.Context) error {
    // Verificar se servidor suporta IDLE
    caps := s.imapClient.Capabilities()
    if !caps.Has("IDLE") {
        s.logger.Warn("Server does not support IDLE, falling back to polling")
        return s.StartPollingSync(ctx)
    }

    // Callback para updates
    onUpdate := func(update imap.IDLEUpdate) {
        switch update.Type {
        case imap.UpdateNewMail:
            s.logger.Info("New mail detected", "mailbox", update.Mailbox)
            // Sync apenas novos emails
            go s.syncNewEmails(ctx, update.Mailbox)
            // Emitir evento
            s.eventBus.Publish(NewEmailDetectedEvent{
                Mailbox:  update.Mailbox,
                NewCount: update.NewCount,
            })

        case imap.UpdateExpunge:
            s.logger.Info("Email deleted on server", "seqnum", update.SeqNum)
            // Sync deleções
            go s.syncDeletions(ctx, update.Mailbox)
        }
    }

    // Iniciar IDLE no INBOX
    return s.imapClient.StartIDLE(ctx, "INBOX", onUpdate)
}

// Fallback para polling se IDLE não disponível
func (s *SyncService) StartPollingSync(ctx context.Context) error {
    ticker := time.NewTicker(s.config.SyncInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := s.SyncAll(ctx); err != nil {
                s.logger.Error("Sync failed", "error", err)
            }
        }
    }
}
```

### 3. Application - Start IDLE

Atualizar `internal/app/app.go`:

```go
func (a *Application) Start(ctx context.Context) error {
    // ... código existente ...

    // Iniciar sync em tempo real
    go func() {
        if err := a.syncService.StartRealtimeSync(ctx); err != nil {
            a.logger.Error("Realtime sync failed", "error", err)
            // Fallback para polling
            a.syncService.StartPollingSync(ctx)
        }
    }()

    return nil
}
```

### 4. Event para UI

```go
// internal/ports/events.go
type NewEmailDetectedEvent struct {
    Mailbox  string
    NewCount uint32
}

// Desktop deve ouvir este evento e atualizar UI
// TUI deve ouvir e mostrar notificação
```

### 5. Desktop - Realtime Updates

```javascript
// frontend/src/lib/stores/emails.js
import { EventsOn } from '../wailsjs/runtime/runtime';

// Ouvir eventos de novos emails
EventsOn('email:detected', ({ mailbox, newCount }) => {
    // Recarregar lista se no mailbox atual
    if (currentFolder === mailbox) {
        loadEmails();
    }
    // Mostrar notificação
    showNotification(`${newCount} novo(s) email(s)`);
});
```

## Cuidados

1. **Timeout**: RFC recomenda < 29 minutos
2. **Reconnect**: Tratar desconexões gracefully
3. **Multiple mailboxes**: IDLE só funciona em um mailbox por conexão
4. **Bandwidth**: IDLE usa conexão persistente

## Critérios de Aceitação

- [ ] Novos emails aparecem em < 5 segundos
- [ ] Funciona com Gmail
- [ ] Fallback para polling se IDLE não suportado
- [ ] Reconnect automático em caso de erro
- [ ] Logs claros de estado
- [ ] Não quebra sync manual (tecla r)

## Testes

```go
func TestIDLE(t *testing.T) {
    // 1. Conectar via IDLE
    // 2. Enviar email para conta (via SMTP)
    // 3. Verificar que callback foi chamado em < 10s
    // 4. Verificar que email aparece no banco
}
```

---

*Prompt criado: 2025-12-12*
