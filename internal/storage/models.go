package storage

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

// SQLiteTime handles time parsing from SQLite strings
type SQLiteTime struct {
	time.Time
}

func (t *SQLiteTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		var formats = []string{
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02T15:04:05.999999999-07:00",
			"2006-01-02T15:04:05.999999999Z",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05 -0700 -0700",
			"2006-01-02 15:04:05 -0700 MST",
			"2006-01-02 15:04:05 -0700",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, v); err == nil {
				t.Time = parsed
				return nil
			}
		}
		return fmt.Errorf("cannot parse time: %s", v)
	default:
		return fmt.Errorf("unsupported type for SQLiteTime: %T", value)
	}
}

func (t SQLiteTime) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Time.Format("2006-01-02 15:04:05"), nil
}

type Account struct {
	ID        int64      `db:"id"`
	Email     string     `db:"email"`
	Name      string     `db:"name"`
	CreatedAt SQLiteTime `db:"created_at"`
}

type Folder struct {
	ID             int64          `db:"id"`
	AccountID      int64          `db:"account_id"`
	Name           string         `db:"name"`
	TotalMessages  int            `db:"total_messages"`
	UnreadMessages int            `db:"unread_messages"`
	LastSync       sql.NullTime   `db:"last_sync"`
}

type Email struct {
	ID             int64          `db:"id"`
	AccountID      int64          `db:"account_id"`
	FolderID       int64          `db:"folder_id"`
	UID            uint32         `db:"uid"`
	MessageID      sql.NullString `db:"message_id"`
	Subject        string         `db:"subject"`
	FromName       string         `db:"from_name"`
	FromEmail      string         `db:"from_email"`
	ToAddresses    string         `db:"to_addresses"`
	CcAddresses    string         `db:"cc_addresses"`
	Date           SQLiteTime     `db:"date"`
	IsRead         bool           `db:"is_read"`
	IsStarred      bool           `db:"is_starred"`
	IsArchived     bool           `db:"is_archived"`
	IsDeleted      bool           `db:"is_deleted"`
	IsReplied      bool           `db:"is_replied"`
	HasAttachments bool           `db:"has_attachments"`
	Snippet        string         `db:"snippet"`
	BodyText       string         `db:"body_text"`
	BodyHTML       string         `db:"body_html"`
	RawHeaders     string         `db:"raw_headers"`
	Size           int64          `db:"size"`
	BodyIndexed    bool           `db:"body_indexed"`
	InReplyTo      sql.NullString `db:"in_reply_to"`
	References     sql.NullString `db:"references"`
	ThreadID       sql.NullString `db:"thread_id"`
	CreatedAt      SQLiteTime     `db:"created_at"`
	UpdatedAt      SQLiteTime     `db:"updated_at"`
}

// EmailSummary é uma versão resumida para listagem
type EmailSummary struct {
	ID             int64          `db:"id"`
	UID            uint32         `db:"uid"`
	MessageID      sql.NullString `db:"message_id"`
	Subject        string         `db:"subject"`
	FromName       string         `db:"from_name"`
	FromEmail      string         `db:"from_email"`
	Date           SQLiteTime     `db:"date"`
	IsRead         bool           `db:"is_read"`
	IsStarred      bool           `db:"is_starred"`
	IsReplied      bool           `db:"is_replied"`
	HasAttachments bool           `db:"has_attachments"`
	Snippet        string         `db:"snippet"`
	ThreadID       sql.NullString `db:"thread_id"`
}

// DraftStatus representa o estado de um draft
type DraftStatus string

const (
	DraftStatusDraft     DraftStatus = "draft"     // Aguardando aprovação (AI drafts)
	DraftStatusScheduled DraftStatus = "scheduled" // Aprovado, aguardando delay para envio
	DraftStatusSending   DraftStatus = "sending"   // Em processo de envio
	DraftStatusSent      DraftStatus = "sent"      // Enviado com sucesso
	DraftStatusCancelled DraftStatus = "cancelled" // Cancelado pelo usuário
	DraftStatusFailed    DraftStatus = "failed"    // Falha no envio
)

// Draft representa um rascunho ou email agendado para envio
type Draft struct {
	ID               int64          `db:"id"`
	AccountID        int64          `db:"account_id"`
	ToAddresses      string         `db:"to_addresses"`
	CcAddresses      sql.NullString `db:"cc_addresses"`
	BccAddresses     sql.NullString `db:"bcc_addresses"`
	Subject          string         `db:"subject"`
	BodyHTML         sql.NullString `db:"body_html"`
	BodyText         sql.NullString `db:"body_text"`
	Classification   sql.NullString `db:"classification"`
	InReplyTo        sql.NullString `db:"in_reply_to"`
	ReferenceIDs     sql.NullString `db:"reference_ids"`
	ReplyToEmailID   sql.NullInt64  `db:"reply_to_email_id"`
	Status           DraftStatus    `db:"status"`
	ScheduledSendAt  sql.NullTime   `db:"scheduled_send_at"`
	SentAt           sql.NullTime   `db:"sent_at"`
	GenerationSource string         `db:"generation_source"` // "manual" ou "ai"
	AIPrompt         sql.NullString `db:"ai_prompt"`
	ErrorMessage     sql.NullString `db:"error_message"`
	CreatedAt        SQLiteTime     `db:"created_at"`
	UpdatedAt        SQLiteTime     `db:"updated_at"`
}

// === ARCHIVE TABLES (permanent storage - never delete) ===

// EmailArchive armazena emails permanentemente após remoção do servidor
type EmailArchive struct {
	ID                int64          `db:"id"`
	OriginalID        int64          `db:"original_id"`
	AccountID         int64          `db:"account_id"`
	FolderID          int64          `db:"folder_id"`
	UID               uint32         `db:"uid"`
	MessageID         sql.NullString `db:"message_id"`
	Subject           string         `db:"subject"`
	FromName          string         `db:"from_name"`
	FromEmail         string         `db:"from_email"`
	ToAddresses       string         `db:"to_addresses"`
	CcAddresses       string         `db:"cc_addresses"`
	Date              SQLiteTime     `db:"date"`
	IsRead            bool           `db:"is_read"`
	IsStarred         bool           `db:"is_starred"`
	HasAttachments    bool           `db:"has_attachments"`
	Snippet           string         `db:"snippet"`
	BodyText          string         `db:"body_text"`
	BodyHTML          string         `db:"body_html"`
	RawHeaders        string         `db:"raw_headers"`
	Size              int64          `db:"size"`
	OriginalCreatedAt SQLiteTime     `db:"original_created_at"`
	OriginalUpdatedAt SQLiteTime     `db:"original_updated_at"`
	ArchivedAt        SQLiteTime     `db:"archived_at"`
	ArchiveReason     string         `db:"archive_reason"` // server_purged, user_deleted, manual_archive
}

// DraftHistory armazena histórico permanente de drafts
type DraftHistory struct {
	ID                int64        `db:"id"`
	OriginalID        int64        `db:"original_id"`
	AccountID         int64        `db:"account_id"`
	ToAddresses       string       `db:"to_addresses"`
	CcAddresses       string       `db:"cc_addresses"`
	BccAddresses      string       `db:"bcc_addresses"`
	Subject           string       `db:"subject"`
	BodyHTML          string       `db:"body_html"`
	BodyText          string       `db:"body_text"`
	Classification    string       `db:"classification"`
	InReplyTo         string       `db:"in_reply_to"`
	ReferenceIDs      string       `db:"reference_ids"`
	ReplyToEmailID    sql.NullInt64 `db:"reply_to_email_id"`
	FinalStatus       string       `db:"final_status"` // sent, cancelled, deleted, failed
	ScheduledSendAt   sql.NullTime `db:"scheduled_send_at"`
	SentAt            sql.NullTime `db:"sent_at"`
	GenerationSource  string       `db:"generation_source"`
	AIPrompt          string       `db:"ai_prompt"`
	ErrorMessage      string       `db:"error_message"`
	OriginalCreatedAt SQLiteTime   `db:"original_created_at"`
	OriginalUpdatedAt SQLiteTime   `db:"original_updated_at"`
	ArchivedAt        SQLiteTime   `db:"archived_at"`
}

// SentEmail registro permanente de emails enviados
type SentEmail struct {
	ID             int64          `db:"id"`
	AccountID      int64          `db:"account_id"`
	MessageID      sql.NullString `db:"message_id"`
	ToAddresses    string         `db:"to_addresses"`
	CcAddresses    string         `db:"cc_addresses"`
	BccAddresses   string         `db:"bcc_addresses"`
	Subject        string         `db:"subject"`
	BodyHTML       string         `db:"body_html"`
	BodyText       string         `db:"body_text"`
	InReplyTo      string         `db:"in_reply_to"`
	ReferenceIDs   string         `db:"reference_ids"`
	ReplyToEmailID sql.NullInt64  `db:"reply_to_email_id"`
	SentAt         SQLiteTime     `db:"sent_at"`
	SendMethod     string         `db:"send_method"` // smtp, gmail_api
	DraftID        sql.NullInt64  `db:"draft_id"`
}

// PendingBatchOp representa uma operação em lote aguardando confirmação
type PendingBatchOp struct {
	ID          int64        `db:"id"`
	AccountID   int64        `db:"account_id"`
	Operation   string       `db:"operation"`    // archive, delete, mark_read, mark_unread
	Description string       `db:"description"`  // "Arquivar 15 emails de newsletter@example.com"
	FilterQuery string       `db:"filter_query"` // descrição do filtro
	EmailIDs    string       `db:"email_ids"`    // JSON array de IDs
	EmailCount  int            `db:"email_count"`
	PreviewData sql.NullString `db:"preview_data"` // JSON com preview (pode ser NULL)
	Status      string         `db:"status"`       // pending, confirmed, cancelled, executed
	CreatedAt   SQLiteTime   `db:"created_at"`
	ExecutedAt  sql.NullTime `db:"executed_at"`
}

// EmailPreview para exibição no preview de operações
type EmailPreview struct {
	ID        int64  `json:"id"`
	Subject   string `json:"subject"`
	FromName  string `json:"from_name"`
	FromEmail string `json:"from_email"`
	Date      string `json:"date"`
}

// ContentIndexState estado do indexador de conteúdo em background
type ContentIndexState struct {
	ID             int64          `db:"id"`
	AccountID      int64          `db:"account_id"`
	Status         string         `db:"status"` // idle, running, paused, completed, error
	TotalEmails    int            `db:"total_emails"`
	IndexedEmails  int            `db:"indexed_emails"`
	LastIndexedUID int64          `db:"last_indexed_uid"`
	Speed          int            `db:"speed"` // emails por minuto
	LastError      sql.NullString `db:"last_error"`
	StartedAt      sql.NullTime   `db:"started_at"`
	PausedAt       sql.NullTime   `db:"paused_at"`
	CompletedAt    sql.NullTime   `db:"completed_at"`
	CreatedAt      SQLiteTime     `db:"created_at"`
	UpdatedAt      SQLiteTime     `db:"updated_at"`
}

// AppSetting configuração do app por conta
type AppSetting struct {
	ID        int64      `db:"id"`
	AccountID int64      `db:"account_id"`
	Key       string     `db:"key"`
	Value     string     `db:"value"`
	CreatedAt SQLiteTime `db:"created_at"`
	UpdatedAt SQLiteTime `db:"updated_at"`
}

// IndexStatus constantes de status do indexador
const (
	IndexStatusIdle      = "idle"
	IndexStatusRunning   = "running"
	IndexStatusPaused    = "paused"
	IndexStatusCompleted = "completed"
	IndexStatusError     = "error"
)

// === ATTACHMENTS ===

// Attachment representa um anexo de email
type Attachment struct {
	ID                 int64          `db:"id"`
	EmailID            int64          `db:"email_id"`
	AccountID          int64          `db:"account_id"`
	Filename           string         `db:"filename"`
	ContentType        string         `db:"content_type"`
	ContentID          sql.NullString `db:"content_id"`
	ContentDisposition sql.NullString `db:"content_disposition"`
	PartNumber         sql.NullString `db:"part_number"`
	Size               int64          `db:"size"`
	Checksum           sql.NullString `db:"checksum"`
	Encoding           sql.NullString `db:"encoding"`
	Charset            sql.NullString `db:"charset"`
	IsInline           bool           `db:"is_inline"`
	IsDownloaded       bool           `db:"is_downloaded"`
	IsCached           bool           `db:"is_cached"`
	CachePath          sql.NullString `db:"cache_path"`
	CachedAt           sql.NullTime   `db:"cached_at"`
	CreatedAt          SQLiteTime     `db:"created_at"`
}

// AttachmentSummary versão resumida para listagem
type AttachmentSummary struct {
	ID          int64  `db:"id"`
	EmailID     int64  `db:"email_id"`
	Filename    string `db:"filename"`
	ContentType string `db:"content_type"`
	Size        int64  `db:"size"`
	IsInline    bool   `db:"is_inline"`
	IsCached    bool   `db:"is_cached"`
	PartNumber  string `db:"part_number"`
}

// === CONTACTS ===

// Contact representa um contato sincronizado do Google People API
type Contact struct {
	ID                  int64          `db:"id"`
	AccountID           int64          `db:"account_id"`
	ResourceName        string         `db:"resource_name"` // people/c1234567890
	DisplayName         string         `db:"display_name"`
	GivenName           sql.NullString `db:"given_name"`
	FamilyName          sql.NullString `db:"family_name"`
	PhotoURL            sql.NullString `db:"photo_url"`
	PhotoETag           sql.NullString `db:"photo_etag"`
	PhotoPath           sql.NullString `db:"photo_path"` // local cache path
	IsStarred           bool           `db:"is_starred"`
	InteractionCount    int            `db:"interaction_count"`
	LastInteractionAt   sql.NullTime   `db:"last_interaction_at"`
	MetadataJSON        sql.NullString `db:"metadata_json"`
	SyncedAt            sql.NullTime   `db:"synced_at"`
	CreatedAt           SQLiteTime     `db:"created_at"`
	UpdatedAt           SQLiteTime     `db:"updated_at"`
}

// ContactEmail representa um endereço de email associado a um contato
type ContactEmail struct {
	ID        int64      `db:"id"`
	ContactID int64      `db:"contact_id"`
	Email     string     `db:"email"`
	EmailType string     `db:"email_type"` // home, work, other
	IsPrimary bool       `db:"is_primary"`
	CreatedAt SQLiteTime `db:"created_at"`
}

// ContactPhone representa um telefone associado a um contato
type ContactPhone struct {
	ID          int64      `db:"id"`
	ContactID   int64      `db:"contact_id"`
	PhoneNumber string     `db:"phone_number"`
	PhoneType   string     `db:"phone_type"` // mobile, work, home, other
	IsPrimary   bool       `db:"is_primary"`
	CreatedAt   SQLiteTime `db:"created_at"`
}

// ContactInteraction representa uma interação (email enviado/recebido) com um contato
type ContactInteraction struct {
	ID              int64          `db:"id"`
	ContactID       int64          `db:"contact_id"`
	EmailID         sql.NullInt64  `db:"email_id"`
	InteractionType string         `db:"interaction_type"` // received, sent
	InteractionDate SQLiteTime     `db:"interaction_date"`
	CreatedAt       SQLiteTime     `db:"created_at"`
}

// ContactsSyncState rastreia o estado do sync de contatos
type ContactsSyncState struct {
	ID                   int64          `db:"id"`
	AccountID            int64          `db:"account_id"`
	LastSyncToken        sql.NullString `db:"last_sync_token"`
	LastFullSync         sql.NullTime   `db:"last_full_sync"`
	LastIncrementalSync  sql.NullTime   `db:"last_incremental_sync"`
	TotalContacts        int            `db:"total_contacts"`
	Status               string         `db:"status"` // never_synced, syncing, synced, error
	ErrorMessage         sql.NullString `db:"error_message"`
	CreatedAt            SQLiteTime     `db:"created_at"`
	UpdatedAt            SQLiteTime     `db:"updated_at"`
}

// ContactsSyncStatus constantes de status do sync
const (
	ContactsSyncNeverSynced = "never_synced"
	ContactsSyncSyncing     = "syncing"
	ContactsSyncSynced      = "synced"
	ContactsSyncError       = "error"
)

// ContactWithEmails representa um contato com seus emails para facilitar queries
type ContactWithEmails struct {
	Contact
	Emails []ContactEmail `db:"-"`
	Phones []ContactPhone `db:"-"`
}
