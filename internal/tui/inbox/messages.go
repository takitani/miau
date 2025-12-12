package inbox

import (
	"time"

	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/storage"
)

// Messages for tea.Cmd communication

type dbInitMsg struct{}

type connectedMsg struct {
	client *imap.Client
}

type foldersLoadedMsg struct {
	mailboxes []imap.Mailbox
}

type syncProgressMsg struct {
	status string
	synced int
	total  int
}

type syncDoneMsg struct {
	synced   int
	total    int
	purged   int
	archived int // emails movidos para arquivo permanente
}

type emailsLoadedMsg struct {
	emails []storage.EmailSummary
}

type errMsg struct {
	err error
}

type configSavedMsg struct{}

type aiResponseMsg struct {
	response string
	err      error
}

type htmlOpenedMsg struct {
	err error
}

type emailContentMsg struct {
	content string
	err     error
}

type aiEmailContextMsg struct {
	email   *storage.EmailSummary
	content string
	err     error
}

type emailSentMsg struct {
	err     error
	host    string
	port    int
	to      string
	msgID   string
	backend string // "smtp" ou "gmail_api"
}

type markReadMsg struct {
	emailID int64
	uid     uint32
}

type debugLogMsg struct {
	msg string
}

type bounceCheckTickMsg struct{}

type bounceFoundMsg struct {
	originalTo      string
	originalSubject string
	bounceReason    string
	bounceFrom      string
	bounceSubject   string
}

// Draft messages
type draftCreatedMsg struct {
	draft *storage.Draft
	err   error
}

type draftScheduledMsg struct {
	draft  *storage.Draft
	sendAt time.Time
	err    error
}

type draftSentMsg struct {
	draftID int64
	to      string
	backend string
	err     error
}

type draftSendTickMsg struct{}

// Snooze tick
type snoozeTickMsg struct{}

// Auto-refresh messages
type autoRefreshTickMsg struct{}

type draftsLoadedMsg struct {
	drafts    []storage.Draft
	err       error
	accountID int64
}

// Archive/Delete messages
type emailArchivedMsg struct {
	emailID int64
	err     error
}

type emailDeletedMsg struct {
	emailID int64
	err     error
}

// Batch operation filter messages
type batchFilterAppliedMsg struct {
	op     *storage.PendingBatchOp
	emails []storage.EmailSummary
	err    error
}

type batchOpExecutedMsg struct {
	count int
	err   error
}

type checkPendingBatchOpsMsg struct {
	op  *storage.PendingBatchOp
	err error
}

// Search messages
type searchResultsMsg struct {
	results []storage.EmailSummary
	query   string
	err     error
}

// Analytics messages
type analyticsLoadedMsg struct {
	data *AnalyticsData
	err  error
}

type searchDebounceMsg struct {
	query string
}

// Settings and Indexer messages
type indexStateLoadedMsg struct {
	state *storage.ContentIndexState
	err   error
}

type indexerTickMsg struct{}

type indexBatchDoneMsg struct {
	indexed int
	lastUID int64
	err     error
}

// Image preview messages
type imageAttachmentsMsg struct {
	attachments []Attachment
	err         error
}

type imageRenderedMsg struct {
	output string
	err    error
}

type imageSavedMsg struct {
	path string
	err  error
}

// All attachments messages
type allAttachmentsMsg struct {
	attachments []Attachment
	err         error
}

type attachmentSavedMsg struct {
	path string
	err  error
}

type desktopLaunchedMsg struct {
	success bool
	err     error
}

type settingsFoldersLoadedMsg struct {
	folders []SettingsFolder
	err     error
}

type settingsSavedMsg struct {
	err error
}

// Account switch message
type accountSwitchedMsg struct {
	email string
	err   error
}
