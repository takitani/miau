# Threading Implementation Plan

## ✅ Phase 1: Backend Infrastructure (COMPLETED)

### Database Schema
- ✅ Added `in_reply_to`, `references`, `thread_id` columns to `emails` table
- ✅ Created indexes for fast thread queries
- ✅ Migration system handles existing databases

### Thread Detection Algorithm
```
GenerateThreadID(messageID, inReplyTo, references, subject):
  1. If In-Reply-To exists → use parent's Message-ID
  2. Else if References exists → use first Message-ID (thread root)
  3. Else if MessageID exists → use own (new thread)
  4. Fallback → use normalized subject (removes Re:, Fwd:, etc)
```

### Storage Layer (internal/storage/)
- ✅ `DetectAndUpdateThreadID()` - Auto-populates thread_id
- ✅ `GetThreadEmails()` - Returns all emails in thread (DESC by date)
- ✅ `GetThreadForEmail()` - Gets thread for specific email
- ✅ `GetThreadSummaries()` - Groups emails by thread for inbox
- ✅ `CountThreadEmails()` - Count messages in thread
- ✅ `GetThreadParticipants()` - Get unique senders in thread

### Integration
- ✅ IMAP extracts In-Reply-To & References from Envelope
- ✅ Sync flow populates threading fields automatically
- ✅ Thread detection runs on every email insert/update

---

## 🚧 Phase 2: Service Layer (TODO)

### Thread Service (internal/services/thread.go)

**CRITICAL**: Following REGRA DE OURO - NUNCA duplicar lógica!

```go
type ThreadService struct {
    storage ports.StoragePort
    events  ports.EventBus
}

// GetThread returns full thread with all messages
func (s *ThreadService) GetThread(ctx context.Context, emailID int64) (*ports.Thread, error)

// GetThreadSummary returns thread metadata for inbox display
func (s *ThreadService) GetThreadSummary(ctx context.Context, threadID string) (*ports.ThreadSummary, error)

// ExpandThread returns all messages for a collapsed thread
func (s *ThreadService) ExpandThread(ctx context.Context, threadID string) ([]ports.EmailContent, error)
```

### Types (internal/ports/types.go)

```go
type Thread struct {
    ThreadID     string
    Subject      string
    Participants []string
    MessageCount int
    Messages     []EmailContent  // Ordered DESC by date (newest first)
    IsRead       bool            // All messages read?
}

type ThreadSummary struct {
    ThreadID       string
    Subject        string
    LastSender     string
    LastDate       time.Time
    MessageCount   int
    UnreadCount    int
    HasAttachments bool
    Participants   []string
}
```

---

## 🎨 Phase 3: Desktop UI (TODO)

### Design: Hybrid Minimap + Stack View

```
┌────────────────────────────────────────┬──┐
│ Re: Website Redesign (5 messages)     │█│ ← Minimap (20px)
│                                        │ │   Vertical scrollbar
├────────────────────────────────────────┤●│ ← You (2h ago)
│ 📧 From: João Silva                    │ │
│    To: Maria, Pedro                    │ │
│    2 hours ago                         │●│ ← Maria (1h ago)
│                                        │ │
│ E aí pessoal, o que acharam do        │ │
│ mockup? Segue anexo.                  │●│ ← Pedro (45m ago)
│                                        │ │
│ 📎 mockup.pdf                          │ │
├────────────────────────────────────────┤●│ ← You (30m ago)
│ ▸ Maria Santos                         │ │
│   1 hour ago                           │●│ ← Maria (now)
│   "Adorei! Só acho que o header..."    │ │
├────────────────────────────────────────┤ │
│ ▸ Pedro Costa (collapsed)              │ │
│   45 min ago                           │ │
└────────────────────────────────────────┴──┘
```

### Components

**ThreadView.svelte**
- Main container
- Handles minimap + message stack layout
- Manages expand/collapse state

**ThreadMinimap.svelte**
- Vertical bar (20px width)
- Dots represent messages (● ○ ◆)
- Color coded by participant
- Click to jump to message
- Highlight current message

**ThreadMessage.svelte**
- Single message component
- Two states: collapsed/expanded
- Collapsed: shows header + 1 line preview
- Expanded: shows full content
- Smooth transitions

**ThreadHeader.svelte**
- Thread subject
- Participant pills
- Message count badge
- Actions (collapse all, mark read, etc)

### Behavior
- Last message always expanded by default
- Click header to expand/collapse
- Click minimap dot to jump + expand
- Scroll syncs with minimap highlight
- Keyboard: ↑↓ navigate, Enter expand/collapse

---

## 📟 Phase 4: TUI (TODO)

### Design: Similar to Desktop but keyboard-first

```
┌────────────────────────────────────────┬─┐
│ Re: Website Redesign (5 msgs)    [m]  │█│
├────────────────────────────────────────┤●│
│ ▸ João Silva → Maria, Pedro  2h ago   │ │
│   E aí pessoal, o que acharam...       │●│
├────────────────────────────────────────┤ │
│ ▾ Maria Santos → All          1h ago   │●│
│ ────────────────────────────────────── │ │
│ Adorei! Só acho que o header poderia  │●│
│ ser mais destacado. O que acham?       │ │
│                                        │●│
├────────────────────────────────────────┤ │
│ ▸ Pedro Costa → All           45m ago  │ │
└────────────────────────────────────────┴─┘

Keys: j/k=navigate  Enter=expand  m=toggle minimap
      [/]=prev/next participant  t=collapse all
```

### Components (internal/tui/thread/)

**thread.go** - Main thread view model
- State machine: loading → ready
- Tracks current message index
- Manages minimap visibility

**message.go** - Single message component
- Collapsed/expanded rendering
- Syntax highlighting for code blocks
- Attachment indicators

**minimap.go** - Minimap panel
- ASCII art visualization
- Current position indicator
- Participant legend

### Key Bindings
- `j/k` or `↑↓` - Navigate messages
- `Enter` or `Space` - Expand/collapse current
- `m` - Toggle minimap panel
- `[` / `]` - Jump to prev/next participant
- `t` - Collapse all messages
- `r` - Reply to thread
- `q` or `Esc` - Back to inbox

### Integration with Inbox
- Thread icon `[3]` shows message count in inbox
- Pressing Enter on thread opens thread view
- Unread badge shows unread count `[●2]`

---

## 🧪 Phase 5: Testing (TODO)

### Test Cases

**Thread Detection**
- [ ] Simple reply chain (A → B → C)
- [ ] Multiple replies to same message (tree structure)
- [ ] Subject-based threading (no In-Reply-To)
- [ ] Mixed: some with References, some without
- [ ] Gmail conversation threading
- [ ] Outlook threading behavior

**Edge Cases**
- [ ] Thread with 100+ messages (performance)
- [ ] Orphaned replies (parent deleted)
- [ ] Duplicate Message-IDs
- [ ] Malformed References headers
- [ ] Thread split (same subject, different root)

**UI Testing**
- [ ] Minimap scrolling with long threads
- [ ] Expand/collapse animations
- [ ] Keyboard navigation edge cases
- [ ] Mobile responsive (Desktop only)
- [ ] Dark mode rendering

---

## 📊 Performance Considerations

### Database Indexes
```sql
CREATE INDEX idx_emails_thread_id ON emails(thread_id);
CREATE INDEX idx_emails_message_id ON emails(message_id);
CREATE INDEX idx_emails_in_reply_to ON emails(in_reply_to);
```

### Query Optimization
- Use `GetThreadSummaries()` for inbox (grouped query)
- Only load full thread on-demand (lazy loading)
- Cache thread participant list
- Limit initial load to 50 messages per thread

### UI Performance
- Virtual scrolling for threads with 50+ messages
- Lazy render collapsed messages (header only)
- Debounce minimap updates during fast scroll
- Optimize re-renders in Desktop (React.memo)

---

## 🎯 Success Metrics

1. **Correctness**: 95%+ thread detection accuracy on real mailboxes
2. **Performance**: Load thread of 100 messages in <500ms
3. **UX**: Users prefer thread view over flat list (A/B test)
4. **Visual Clarity**: No "salad" effect (clean, scannable)

---

## 📝 Notes

- **Newest First**: All thread displays show newest message at top
- **Minimap is Optional**: Can be hidden on small screens
- **No Gmail-style Hiding**: All messages visible, just collapsed
- **Consistent UX**: Desktop and TUI follow same patterns
