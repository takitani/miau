package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// === ACCOUNTS ===

func GetOrCreateAccount(email, name string) (*Account, error) {
	var account Account

	err := db.Get(&account, "SELECT * FROM accounts WHERE email = ?", email)
	if err == nil {
		return &account, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Cria nova conta
	var result, err2 = db.Exec("INSERT INTO accounts (email, name) VALUES (?, ?)", email, name)
	if err2 != nil {
		return nil, err2
	}

	var id, _ = result.LastInsertId()
	account = Account{
		ID:        id,
		Email:     email,
		Name:      name,
		CreatedAt: SQLiteTime{time.Now()},
	}

	return &account, nil
}

// === FOLDERS ===

func GetOrCreateFolder(accountID int64, name string) (*Folder, error) {
	var folder Folder

	err := db.Get(&folder, "SELECT * FROM folders WHERE account_id = ? AND name = ?", accountID, name)
	if err == nil {
		return &folder, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Cria nova pasta
	var result, err2 = db.Exec("INSERT INTO folders (account_id, name) VALUES (?, ?)", accountID, name)
	if err2 != nil {
		return nil, err2
	}

	var id, _ = result.LastInsertId()
	folder = Folder{
		ID:        id,
		AccountID: accountID,
		Name:      name,
	}

	return &folder, nil
}

func GetFolders(accountID int64) ([]Folder, error) {
	var folders []Folder
	err := db.Select(&folders, "SELECT * FROM folders WHERE account_id = ? ORDER BY name", accountID)
	return folders, err
}

func UpdateFolderStats(folderID int64, total, unread int) error {
	_, err := db.Exec(`
		UPDATE folders
		SET total_messages = ?, unread_messages = ?, last_sync = CURRENT_TIMESTAMP
		WHERE id = ?`,
		total, unread, folderID)
	return err
}

// === EMAILS ===

func UpsertEmail(e *Email) error {
	_, err := db.Exec(`
		INSERT INTO emails (
			account_id, folder_id, uid, message_id, subject,
			from_name, from_email, to_addresses, cc_addresses, date,
			is_read, is_starred, is_deleted, has_attachments, snippet,
			body_text, body_html, raw_headers, size, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(account_id, folder_id, uid) DO UPDATE SET
			subject = excluded.subject,
			from_name = excluded.from_name,
			from_email = excluded.from_email,
			is_read = excluded.is_read,
			is_starred = excluded.is_starred,
			is_deleted = excluded.is_deleted,
			body_text = excluded.body_text,
			body_html = excluded.body_html,
			updated_at = CURRENT_TIMESTAMP`,
		e.AccountID, e.FolderID, e.UID, e.MessageID, e.Subject,
		e.FromName, e.FromEmail, e.ToAddresses, e.CcAddresses, e.Date,
		e.IsRead, e.IsStarred, e.IsDeleted, e.HasAttachments, e.Snippet,
		e.BodyText, e.BodyHTML, e.RawHeaders, e.Size)
	return err
}

func GetEmails(accountID, folderID int64, limit, offset int) ([]EmailSummary, error) {
	var emails []EmailSummary
	var err error

	// If accountID is 0, search by folderID only (folderID is unique)
	if accountID == 0 {
		err = db.Select(&emails, `
			SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
			FROM emails
			WHERE folder_id = ? AND is_archived = 0 AND is_deleted = 0
			ORDER BY date DESC
			LIMIT ? OFFSET ?`,
			folderID, limit, offset)
	} else {
		err = db.Select(&emails, `
			SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
			FROM emails
			WHERE account_id = ? AND folder_id = ? AND is_archived = 0 AND is_deleted = 0
			ORDER BY date DESC
			LIMIT ? OFFSET ?`,
			accountID, folderID, limit, offset)
	}
	return emails, err
}

func GetEmailByID(id int64) (*Email, error) {
	var email Email
	err := db.Get(&email, "SELECT * FROM emails WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

// EmailWithFolder contains email data and folder name
type EmailWithFolder struct {
	Email
	FolderName string `db:"folder_name"`
}

// GetEmailByIDWithFolder returns email with folder name (needed for IMAP fetch)
func GetEmailByIDWithFolder(id int64) (*EmailWithFolder, error) {
	var email EmailWithFolder
	err := db.Get(&email, `
		SELECT e.*, f.name as folder_name
		FROM emails e
		JOIN folders f ON e.folder_id = f.id
		WHERE e.id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

func GetEmailByUID(accountID, folderID int64, uid uint32) (*Email, error) {
	var email Email
	err := db.Get(&email, "SELECT * FROM emails WHERE account_id = ? AND folder_id = ? AND uid = ?", accountID, folderID, uid)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

func EmailExistsByUID(accountID, folderID int64, uid uint32) (bool, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM emails WHERE account_id = ? AND folder_id = ? AND uid = ?", accountID, folderID, uid)
	return count > 0, err
}

func GetLatestUID(accountID, folderID int64) (uint32, error) {
	var uid uint32
	err := db.Get(&uid, "SELECT COALESCE(MAX(uid), 0) FROM emails WHERE account_id = ? AND folder_id = ?", accountID, folderID)
	return uid, err
}

func MarkAsRead(id int64, read bool) error {
	_, err := db.Exec("UPDATE emails SET is_read = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", read, id)
	return err
}

func MarkAsStarred(id int64, starred bool) error {
	_, err := db.Exec("UPDATE emails SET is_starred = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", starred, id)
	return err
}

func MarkAsReplied(id int64) error {
	_, err := db.Exec("UPDATE emails SET is_replied = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func MarkAsArchived(id int64, archived bool) error {
	_, err := db.Exec("UPDATE emails SET is_archived = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", archived, id)
	return err
}

// ArchiveEmailsByFilter arquiva emails por filtro (para uso do AI)
func ArchiveEmailsByFilter(accountID int64, fromEmail string) (int64, error) {
	var result, err = db.Exec(`
		UPDATE emails SET is_archived = 1, updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ? AND from_email LIKE ? AND is_archived = 0`,
		accountID, "%"+fromEmail+"%")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func DeleteEmail(id int64) error {
	_, err := db.Exec("UPDATE emails SET is_deleted = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

// PurgeDeletedFromServer remove emails do banco que não existem mais no servidor
func PurgeDeletedFromServer(accountID, folderID int64, serverUIDs []uint32) (int, error) {
	if len(serverUIDs) == 0 {
		// Se servidor retornou vazio, NÃO deleta nada (pode ser erro de conexão)
		return 0, nil
	}

	// Busca UIDs locais
	var localUIDs []uint32
	var err = db.Select(&localUIDs, `
		SELECT uid FROM emails WHERE account_id = ? AND folder_id = ? AND is_deleted = 0`, accountID, folderID)
	if err != nil {
		return 0, err
	}

	// Cria set de UIDs do servidor para lookup rápido
	var serverSet = make(map[uint32]bool)
	for _, uid := range serverUIDs {
		serverSet[uid] = true
	}

	// Encontra UIDs que existem local mas não no servidor
	var toDelete []uint32
	for _, uid := range localUIDs {
		if !serverSet[uid] {
			toDelete = append(toDelete, uid)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Marca como deletados
	for _, uid := range toDelete {
		db.Exec(`UPDATE emails SET is_deleted = 1, updated_at = CURRENT_TIMESTAMP
			WHERE account_id = ? AND folder_id = ? AND uid = ?`, accountID, folderID, uid)
	}

	return len(toDelete), nil
}

func CountEmails(accountID, folderID int64) (total int, unread int, err error) {
	err = db.Get(&total, "SELECT COUNT(*) FROM emails WHERE account_id = ? AND folder_id = ? AND is_archived = 0 AND is_deleted = 0", accountID, folderID)
	if err != nil {
		return
	}
	err = db.Get(&unread, "SELECT COUNT(*) FROM emails WHERE account_id = ? AND folder_id = ? AND is_archived = 0 AND is_deleted = 0 AND is_read = 0", accountID, folderID)
	return
}

// === SEARCH ===

func SearchEmails(accountID int64, query string, limit int) ([]EmailSummary, error) {
	var emails []EmailSummary
	err := db.Select(&emails, `
		SELECT e.id, e.uid, e.message_id, e.subject, e.from_name, e.from_email, e.date, e.is_read, e.is_starred, e.is_replied, e.snippet
		FROM emails e
		JOIN emails_fts fts ON e.id = fts.rowid
		WHERE e.account_id = ? AND e.is_archived = 0 AND e.is_deleted = 0 AND emails_fts MATCH ?
		ORDER BY e.date DESC
		LIMIT ?`,
		accountID, query, limit)
	return emails, err
}

// FuzzySearchEmails busca emails com fuzzy matching
// Usa FTS5 trigram para queries com 3+ caracteres, LIKE para queries menores
// Busca em: subject, from_name, from_email, body_text, snippet
func FuzzySearchEmails(accountID int64, query string, limit int) ([]EmailSummary, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	var emails []EmailSummary
	var err error

	// Para queries com 3+ caracteres, usa FTS5 trigram
	// Trigram automaticamente faz partial matching (ex: "proj" encontra "projeto")
	if len(query) >= 3 {
		// Escapa caracteres especiais do FTS5 e adiciona * para prefix matching
		var ftsQuery = escapeFTSQuery(query)

		err = db.Select(&emails, `
			SELECT e.id, e.uid, e.message_id, e.subject, e.from_name, e.from_email, e.date, e.is_read, e.is_starred, e.is_replied, e.snippet
			FROM emails e
			JOIN emails_fts fts ON e.id = fts.rowid
			WHERE e.account_id = ? AND e.is_archived = 0 AND e.is_deleted = 0 AND emails_fts MATCH ?
			ORDER BY e.date DESC
			LIMIT ?`,
			accountID, ftsQuery, limit)

		// Se FTS5 encontrou resultados, retorna
		if err == nil && len(emails) > 0 {
			return emails, nil
		}
	}

	// Fallback: busca LIKE em múltiplos campos
	// Mais lento mas funciona para queries curtas e casos edge
	var likePattern = "%" + query + "%"
	err = db.Select(&emails, `
		SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
		FROM emails
		WHERE account_id = ? AND is_archived = 0 AND is_deleted = 0
		AND (
			subject LIKE ? COLLATE NOCASE OR
			from_name LIKE ? COLLATE NOCASE OR
			from_email LIKE ? COLLATE NOCASE OR
			snippet LIKE ? COLLATE NOCASE
		)
		ORDER BY date DESC
		LIMIT ?`,
		accountID, likePattern, likePattern, likePattern, likePattern, limit)

	return emails, err
}

// escapeFTSQuery escapa caracteres especiais do FTS5 e prepara a query
func escapeFTSQuery(query string) string {
	// Remove caracteres que podem quebrar a sintaxe FTS5
	var replacer = strings.NewReplacer(
		"\"", "",
		"'", "",
		"*", "",
		"(", "",
		")", "",
		":", "",
		"-", " ",
		"OR", "or",
		"AND", "and",
		"NOT", "not",
	)
	query = replacer.Replace(query)

	// Divide em palavras e junta com OR para fuzzy matching
	var words = strings.Fields(query)
	if len(words) == 0 {
		return query
	}

	// Para múltiplas palavras, usa OR para match parcial
	if len(words) > 1 {
		return strings.Join(words, " OR ")
	}

	return query
}

// GetEmailsToSyncToServer retorna emails que precisam ser sincronizados com o servidor
// (arquivados ou deletados localmente)
func GetEmailsToSyncToServer(accountID, folderID int64) (archived []Email, deleted []Email, err error) {
	err = db.Select(&archived, `
		SELECT * FROM emails
		WHERE account_id = ? AND folder_id = ? AND is_archived = 1`,
		accountID, folderID)
	if err != nil {
		return
	}
	err = db.Select(&deleted, `
		SELECT * FROM emails
		WHERE account_id = ? AND folder_id = ? AND is_deleted = 1`,
		accountID, folderID)
	return
}

// === DRAFTS ===

// CreateDraft cria um novo draft
func CreateDraft(d *Draft) (int64, error) {
	var result, err = db.Exec(`
		INSERT INTO drafts (
			account_id, to_addresses, cc_addresses, bcc_addresses,
			subject, body_html, body_text, classification,
			in_reply_to, reference_ids, reply_to_email_id,
			status, scheduled_send_at, generation_source, ai_prompt
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.AccountID, d.ToAddresses, d.CcAddresses, d.BccAddresses,
		d.Subject, d.BodyHTML, d.BodyText, d.Classification,
		d.InReplyTo, d.ReferenceIDs, d.ReplyToEmailID,
		d.Status, d.ScheduledSendAt, d.GenerationSource, d.AIPrompt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateDraft atualiza um draft existente
func UpdateDraft(d *Draft) error {
	_, err := db.Exec(`
		UPDATE drafts SET
			to_addresses = ?, cc_addresses = ?, bcc_addresses = ?,
			subject = ?, body_html = ?, body_text = ?, classification = ?,
			in_reply_to = ?, reference_ids = ?,
			status = ?, scheduled_send_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		d.ToAddresses, d.CcAddresses, d.BccAddresses,
		d.Subject, d.BodyHTML, d.BodyText, d.Classification,
		d.InReplyTo, d.ReferenceIDs,
		d.Status, d.ScheduledSendAt,
		d.ID)
	return err
}

// DeleteDraft remove um draft permanentemente
func DeleteDraft(id int64) error {
	_, err := db.Exec("DELETE FROM drafts WHERE id = ?", id)
	return err
}

// GetDraftByID busca um draft por ID
func GetDraftByID(id int64) (*Draft, error) {
	var draft Draft
	err := db.Get(&draft, "SELECT * FROM drafts WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &draft, nil
}

// GetDraftsByStatus busca drafts por status
func GetDraftsByStatus(accountID int64, status DraftStatus) ([]Draft, error) {
	var drafts []Draft
	err := db.Select(&drafts, `
		SELECT * FROM drafts
		WHERE account_id = ? AND status = ?
		ORDER BY created_at DESC`,
		accountID, status)
	return drafts, err
}

// GetPendingDrafts busca drafts pendentes (draft ou scheduled)
func GetPendingDrafts(accountID int64) ([]Draft, error) {
	var drafts []Draft
	err := db.Select(&drafts, `
		SELECT * FROM drafts
		WHERE account_id = ? AND status IN ('draft', 'scheduled')
		ORDER BY created_at DESC`,
		accountID)
	return drafts, err
}

// GetScheduledDraftsReady busca drafts agendados prontos para envio
func GetScheduledDraftsReady() ([]Draft, error) {
	var drafts []Draft
	err := db.Select(&drafts, `
		SELECT * FROM drafts
		WHERE status = 'scheduled' AND scheduled_send_at <= CURRENT_TIMESTAMP
		ORDER BY scheduled_send_at ASC`)
	return drafts, err
}

// ScheduleDraft agenda um draft para envio
func ScheduleDraft(id int64, sendAt time.Time) error {
	_, err := db.Exec(`
		UPDATE drafts SET
			status = 'scheduled',
			scheduled_send_at = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		sendAt, id)
	return err
}

// CancelDraft cancela um draft agendado (volta para status draft)
func CancelDraft(id int64) error {
	_, err := db.Exec(`
		UPDATE drafts SET
			status = 'draft',
			scheduled_send_at = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

// MarkDraftSending marca draft como em envio
func MarkDraftSending(id int64) error {
	_, err := db.Exec(`
		UPDATE drafts SET
			status = 'sending',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

// MarkDraftSent marca draft como enviado
func MarkDraftSent(id int64) error {
	_, err := db.Exec(`
		UPDATE drafts SET
			status = 'sent',
			sent_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, id)
	return err
}

// MarkDraftFailed marca draft como falha
func MarkDraftFailed(id int64, errorMsg string) error {
	_, err := db.Exec(`
		UPDATE drafts SET
			status = 'failed',
			error_message = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, errorMsg, id)
	return err
}

// CountPendingDrafts conta drafts pendentes
func CountPendingDrafts(accountID int64) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM drafts
		WHERE account_id = ? AND status IN ('draft', 'scheduled')`,
		accountID)
	return count, err
}

// === ARCHIVE FUNCTIONS (permanent storage - never delete) ===

// ArchiveEmail move um email para o arquivo permanente
func ArchiveEmailPermanently(emailID int64, reason string) error {
	// Busca o email completo
	var email Email
	var err = db.Get(&email, "SELECT * FROM emails WHERE id = ?", emailID)
	if err != nil {
		return err
	}

	// Insere no arquivo permanente
	_, err = db.Exec(`
		INSERT INTO emails_archive (
			original_id, account_id, folder_id, uid, message_id, subject,
			from_name, from_email, to_addresses, cc_addresses, date,
			is_read, is_starred, has_attachments, snippet,
			body_text, body_html, raw_headers, size,
			original_created_at, original_updated_at, archive_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		email.ID, email.AccountID, email.FolderID, email.UID, email.MessageID, email.Subject,
		email.FromName, email.FromEmail, email.ToAddresses, email.CcAddresses, email.Date,
		email.IsRead, email.IsStarred, email.HasAttachments, email.Snippet,
		email.BodyText, email.BodyHTML, email.RawHeaders, email.Size,
		email.CreatedAt, email.UpdatedAt, reason)
	if err != nil {
		return err
	}

	// Remove da tabela principal (agora podemos deletar pois está arquivado)
	_, err = db.Exec("DELETE FROM emails WHERE id = ?", emailID)
	return err
}

// ArchiveDraft move um draft para o histórico permanente
func ArchiveDraftPermanently(draftID int64, finalStatus string) error {
	// Busca o draft completo
	var draft Draft
	var err = db.Get(&draft, "SELECT * FROM drafts WHERE id = ?", draftID)
	if err != nil {
		return err
	}

	// Insere no histórico permanente
	_, err = db.Exec(`
		INSERT INTO drafts_history (
			original_id, account_id, to_addresses, cc_addresses, bcc_addresses,
			subject, body_html, body_text, classification,
			in_reply_to, reference_ids, reply_to_email_id,
			final_status, scheduled_send_at, sent_at,
			generation_source, ai_prompt, error_message,
			original_created_at, original_updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		draft.ID, draft.AccountID, draft.ToAddresses, draft.CcAddresses, draft.BccAddresses,
		draft.Subject, draft.BodyHTML, draft.BodyText, draft.Classification,
		draft.InReplyTo, draft.ReferenceIDs, draft.ReplyToEmailID,
		finalStatus, draft.ScheduledSendAt, draft.SentAt,
		draft.GenerationSource, draft.AIPrompt, draft.ErrorMessage,
		draft.CreatedAt, draft.UpdatedAt)
	if err != nil {
		return err
	}

	// Remove da tabela principal (agora está arquivado)
	_, err = db.Exec("DELETE FROM drafts WHERE id = ?", draftID)
	return err
}

// RecordSentEmail registra um email enviado permanentemente
func RecordSentEmail(accountID int64, messageID, to, cc, bcc, subject, bodyHTML, bodyText, inReplyTo, references, sendMethod string, replyToEmailID, draftID sql.NullInt64) (int64, error) {
	var result, err = db.Exec(`
		INSERT INTO sent_emails (
			account_id, message_id, to_addresses, cc_addresses, bcc_addresses,
			subject, body_html, body_text, in_reply_to, reference_ids,
			reply_to_email_id, send_method, draft_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		accountID, messageID, to, cc, bcc, subject, bodyHTML, bodyText,
		inReplyTo, references, replyToEmailID, sendMethod, draftID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetSentEmails busca emails enviados
func GetSentEmails(accountID int64, limit, offset int) ([]SentEmail, error) {
	var emails []SentEmail
	err := db.Select(&emails, `
		SELECT * FROM sent_emails
		WHERE account_id = ?
		ORDER BY sent_at DESC
		LIMIT ? OFFSET ?`,
		accountID, limit, offset)
	return emails, err
}

// GetArchivedEmails busca emails arquivados permanentemente
func GetArchivedEmails(accountID int64, limit, offset int) ([]EmailArchive, error) {
	var emails []EmailArchive
	err := db.Select(&emails, `
		SELECT * FROM emails_archive
		WHERE account_id = ?
		ORDER BY date DESC
		LIMIT ? OFFSET ?`,
		accountID, limit, offset)
	return emails, err
}

// GetDraftHistory busca histórico de drafts
func GetDraftHistory(accountID int64, limit, offset int) ([]DraftHistory, error) {
	var drafts []DraftHistory
	err := db.Select(&drafts, `
		SELECT * FROM drafts_history
		WHERE account_id = ?
		ORDER BY archived_at DESC
		LIMIT ? OFFSET ?`,
		accountID, limit, offset)
	return drafts, err
}

// PurgeToArchive move emails deletados há mais de N dias para o arquivo permanente
// Usado para emails que o servidor já purgou (ex: após 30 dias no Trash)
func PurgeToArchive(accountID int64, olderThanDays int) (int, error) {
	// Busca emails deletados há mais de N dias
	var emails []Email
	var err = db.Select(&emails, `
		SELECT * FROM emails
		WHERE account_id = ? AND is_deleted = 1
		AND updated_at < datetime('now', '-' || ? || ' days')`,
		accountID, olderThanDays)
	if err != nil {
		return 0, err
	}

	var count = 0
	for _, email := range emails {
		if err := ArchiveEmailPermanently(email.ID, "server_purged"); err == nil {
			count++
		}
	}
	return count, nil
}

// === BATCH OPERATIONS (preview antes de executar) ===

// CreateBatchOp cria uma operação em lote pendente
func CreateBatchOp(accountID int64, operation, description, filterQuery, emailIDsJSON, previewJSON string, count int) (int64, error) {
	var result, err = db.Exec(`
		INSERT INTO pending_batch_ops (
			account_id, operation, description, filter_query,
			email_ids, email_count, preview_data, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')`,
		accountID, operation, description, filterQuery, emailIDsJSON, count, previewJSON)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetPendingBatchOps retorna operações pendentes
func GetPendingBatchOps(accountID int64) ([]PendingBatchOp, error) {
	var ops []PendingBatchOp
	err := db.Select(&ops, `
		SELECT * FROM pending_batch_ops
		WHERE account_id = ? AND status = 'pending'
		ORDER BY created_at DESC`,
		accountID)
	return ops, err
}

// GetBatchOpByID retorna uma operação por ID
func GetBatchOpByID(id int64) (*PendingBatchOp, error) {
	var op PendingBatchOp
	err := db.Get(&op, "SELECT * FROM pending_batch_ops WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// ConfirmBatchOp marca operação como confirmada
func ConfirmBatchOp(id int64) error {
	_, err := db.Exec(`
		UPDATE pending_batch_ops SET status = 'confirmed'
		WHERE id = ? AND status = 'pending'`, id)
	return err
}

// CancelBatchOp cancela uma operação pendente
func CancelBatchOp(id int64) error {
	_, err := db.Exec(`
		UPDATE pending_batch_ops SET status = 'cancelled'
		WHERE id = ? AND status = 'pending'`, id)
	return err
}

// ExecuteBatchOp executa uma operação confirmada
func ExecuteBatchOp(id int64) (int, error) {
	var op, err = GetBatchOpByID(id)
	if err != nil {
		return 0, err
	}
	if op.Status != "confirmed" && op.Status != "pending" {
		return 0, fmt.Errorf("operação não está pendente ou confirmada")
	}

	// Parse email IDs do JSON
	var emailIDs []int64
	if err := json.Unmarshal([]byte(op.EmailIDs), &emailIDs); err != nil {
		return 0, err
	}

	var count = 0
	for _, emailID := range emailIDs {
		var opErr error
		switch op.Operation {
		case "archive":
			opErr = MarkAsArchived(emailID, true)
		case "delete":
			opErr = DeleteEmail(emailID)
		case "mark_read":
			opErr = MarkAsRead(emailID, true)
		case "mark_unread":
			opErr = MarkAsRead(emailID, false)
		}
		if opErr == nil {
			count++
		}
	}

	// Marca como executada
	db.Exec(`
		UPDATE pending_batch_ops SET status = 'executed', executed_at = CURRENT_TIMESTAMP
		WHERE id = ?`, id)

	return count, nil
}

// PrepareBatchArchive prepara uma operação de arquivamento em lote
// Retorna preview dos emails que serão afetados
func PrepareBatchArchive(accountID int64, fromEmail string) (*PendingBatchOp, error) {
	// Busca emails que serão afetados
	var emails []EmailSummary
	var err = db.Select(&emails, `
		SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
		FROM emails
		WHERE account_id = ? AND from_email LIKE ? AND is_archived = 0 AND is_deleted = 0
		ORDER BY date DESC
		LIMIT 100`,
		accountID, "%"+fromEmail+"%")
	if err != nil {
		return nil, err
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("nenhum email encontrado de '%s'", fromEmail)
	}

	// Prepara IDs e preview
	var emailIDs []int64
	var previews []EmailPreview
	for _, e := range emails {
		emailIDs = append(emailIDs, e.ID)
		previews = append(previews, EmailPreview{
			ID:        e.ID,
			Subject:   e.Subject,
			FromName:  e.FromName,
			FromEmail: e.FromEmail,
			Date:      e.Date.Format("2006-01-02 15:04"),
		})
	}

	var emailIDsJSON, _ = json.Marshal(emailIDs)
	var previewJSON, _ = json.Marshal(previews)

	var description = fmt.Sprintf("Arquivar %d emails de '%s'", len(emails), fromEmail)
	var filterQuery = fmt.Sprintf("from_email LIKE '%%%s%%'", fromEmail)

	var opID, err2 = CreateBatchOp(accountID, "archive", description, filterQuery, string(emailIDsJSON), string(previewJSON), len(emails))
	if err2 != nil {
		return nil, err2
	}

	return GetBatchOpByID(opID)
}

// PrepareBatchDelete prepara uma operação de deleção em lote
func PrepareBatchDelete(accountID int64, fromEmail string) (*PendingBatchOp, error) {
	var emails []EmailSummary
	var err = db.Select(&emails, `
		SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
		FROM emails
		WHERE account_id = ? AND from_email LIKE ? AND is_archived = 0 AND is_deleted = 0
		ORDER BY date DESC
		LIMIT 100`,
		accountID, "%"+fromEmail+"%")
	if err != nil {
		return nil, err
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("nenhum email encontrado de '%s'", fromEmail)
	}

	var emailIDs []int64
	var previews []EmailPreview
	for _, e := range emails {
		emailIDs = append(emailIDs, e.ID)
		previews = append(previews, EmailPreview{
			ID:        e.ID,
			Subject:   e.Subject,
			FromName:  e.FromName,
			FromEmail: e.FromEmail,
			Date:      e.Date.Format("2006-01-02 15:04"),
		})
	}

	var emailIDsJSON, _ = json.Marshal(emailIDs)
	var previewJSON, _ = json.Marshal(previews)

	var description = fmt.Sprintf("Deletar %d emails de '%s'", len(emails), fromEmail)
	var filterQuery = fmt.Sprintf("from_email LIKE '%%%s%%'", fromEmail)

	var opID, err2 = CreateBatchOp(accountID, "delete", description, filterQuery, string(emailIDsJSON), string(previewJSON), len(emails))
	if err2 != nil {
		return nil, err2
	}

	return GetBatchOpByID(opID)
}

// CountPendingBatchOps conta operações pendentes
func CountPendingBatchOps(accountID int64) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM pending_batch_ops
		WHERE account_id = ? AND status = 'pending'`,
		accountID)
	return count, err
}

// GetEmailsByIDs busca emails por lista de IDs (para preview de batch op)
func GetEmailsByIDs(emailIDs []int64) ([]EmailSummary, error) {
	if len(emailIDs) == 0 {
		return nil, nil
	}

	// Constrói query com placeholders
	var placeholders = make([]string, len(emailIDs))
	var args = make([]interface{}, len(emailIDs))
	for i, id := range emailIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	var query = fmt.Sprintf(`
		SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
		FROM emails
		WHERE id IN (%s)
		ORDER BY date DESC`,
		strings.Join(placeholders, ","))

	var emails []EmailSummary
	err := db.Select(&emails, query, args...)
	return emails, err
}

// GetEmailsFiltered busca emails com filtro por remetente
func GetEmailsFiltered(accountID, folderID int64, fromEmailFilter string, limit int) ([]EmailSummary, error) {
	var emails []EmailSummary
	err := db.Select(&emails, `
		SELECT id, uid, message_id, subject, from_name, from_email, date, is_read, is_starred, is_replied, snippet
		FROM emails
		WHERE account_id = ? AND folder_id = ? AND from_email LIKE ? AND is_archived = 0 AND is_deleted = 0
		ORDER BY date DESC
		LIMIT ?`,
		accountID, folderID, "%"+fromEmailFilter+"%", limit)
	return emails, err
}

// === CONTENT INDEXER ===

// GetOrCreateIndexState obtém ou cria estado do indexador para uma conta
func GetOrCreateIndexState(accountID int64) (*ContentIndexState, error) {
	var state ContentIndexState
	err := db.Get(&state, "SELECT * FROM content_index_state WHERE account_id = ?", accountID)
	if err == nil {
		return &state, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Cria novo estado
	_, err = db.Exec(`
		INSERT INTO content_index_state (account_id, status, speed)
		VALUES (?, 'idle', 100)`,
		accountID)
	if err != nil {
		return nil, err
	}

	return GetOrCreateIndexState(accountID) // Recursão segura
}

// UpdateIndexState atualiza o estado do indexador
func UpdateIndexState(accountID int64, status string, indexed int, lastUID int64, lastError string) error {
	var query = `
		UPDATE content_index_state SET
			status = ?,
			indexed_emails = ?,
			last_indexed_uid = ?,
			last_error = NULLIF(?, ''),
			updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ?`

	_, err := db.Exec(query, status, indexed, lastUID, lastError, accountID)
	return err
}

// StartIndexer inicia o indexador
func StartIndexer(accountID int64, totalEmails int) error {
	_, err := db.Exec(`
		UPDATE content_index_state SET
			status = 'running',
			total_emails = ?,
			started_at = CURRENT_TIMESTAMP,
			paused_at = NULL,
			completed_at = NULL,
			last_error = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ?`,
		totalEmails, accountID)
	return err
}

// PauseIndexer pausa o indexador
func PauseIndexer(accountID int64) error {
	_, err := db.Exec(`
		UPDATE content_index_state SET
			status = 'paused',
			paused_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ?`,
		accountID)
	return err
}

// ResumeIndexer retoma o indexador
func ResumeIndexer(accountID int64) error {
	_, err := db.Exec(`
		UPDATE content_index_state SET
			status = 'running',
			paused_at = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ?`,
		accountID)
	return err
}

// CompleteIndexer marca o indexador como completo
func CompleteIndexer(accountID int64) error {
	_, err := db.Exec(`
		UPDATE content_index_state SET
			status = 'completed',
			completed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ?`,
		accountID)
	return err
}

// SetIndexerSpeed define a velocidade do indexador
func SetIndexerSpeed(accountID int64, speed int) error {
	_, err := db.Exec(`
		UPDATE content_index_state SET
			speed = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE account_id = ?`,
		speed, accountID)
	return err
}

// GetEmailsToIndex retorna emails que ainda não foram indexados
func GetEmailsToIndex(accountID int64, limit int) ([]Email, error) {
	var emails []Email
	err := db.Select(&emails, `
		SELECT * FROM emails
		WHERE account_id = ? AND body_indexed = 0 AND is_deleted = 0
		ORDER BY date DESC
		LIMIT ?`,
		accountID, limit)
	return emails, err
}

// MarkEmailIndexed marca um email como indexado
func MarkEmailIndexed(emailID int64, bodyText string) error {
	_, err := db.Exec(`
		UPDATE emails SET
			body_text = ?,
			body_indexed = 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		bodyText, emailID)
	return err
}

// CountEmailsToIndex conta emails não indexados
func CountEmailsToIndex(accountID int64) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM emails
		WHERE account_id = ? AND body_indexed = 0 AND is_deleted = 0`,
		accountID)
	return count, err
}

// CountIndexedEmails conta emails já indexados
func CountIndexedEmails(accountID int64) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM emails
		WHERE account_id = ? AND body_indexed = 1`,
		accountID)
	return count, err
}

// === APP SETTINGS ===

// GetSetting obtém uma configuração
func GetSetting(accountID int64, key string) (string, error) {
	var value string
	err := db.Get(&value, `
		SELECT value FROM app_settings
		WHERE account_id = ? AND key = ?`,
		accountID, key)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting define uma configuração
func SetSetting(accountID int64, key, value string) error {
	_, err := db.Exec(`
		INSERT INTO app_settings (account_id, key, value)
		VALUES (?, ?, ?)
		ON CONFLICT(account_id, key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP`,
		accountID, key, value)
	return err
}

// GetAllSettings obtém todas as configurações de uma conta
func GetAllSettings(accountID int64) (map[string]string, error) {
	var settings []AppSetting
	err := db.Select(&settings, `
		SELECT * FROM app_settings WHERE account_id = ?`,
		accountID)
	if err != nil {
		return nil, err
	}

	var result = make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

// === ANALYTICS ===

// AnalyticsOverviewResult contains general email statistics
type AnalyticsOverviewResult struct {
	TotalEmails    int     `db:"total_emails"`
	UnreadEmails   int     `db:"unread_emails"`
	StarredEmails  int     `db:"starred_emails"`
	ArchivedEmails int     `db:"archived_emails"`
	SentEmails     int     `db:"sent_emails"`
	DraftCount     int     `db:"draft_count"`
	StorageUsedMB  float64 `db:"storage_used_mb"`
}

// GetAnalyticsOverview returns overall email statistics
func GetAnalyticsOverview(accountID int64) (*AnalyticsOverviewResult, error) {
	var overview AnalyticsOverviewResult

	// Total emails (not deleted)
	err := db.Get(&overview.TotalEmails, `
		SELECT COUNT(*) FROM emails WHERE account_id = ? AND is_deleted = 0`,
		accountID)
	if err != nil {
		return nil, err
	}

	// Unread emails
	db.Get(&overview.UnreadEmails, `
		SELECT COUNT(*) FROM emails WHERE account_id = ? AND is_deleted = 0 AND is_archived = 0 AND is_read = 0`,
		accountID)

	// Starred emails
	db.Get(&overview.StarredEmails, `
		SELECT COUNT(*) FROM emails WHERE account_id = ? AND is_deleted = 0 AND is_starred = 1`,
		accountID)

	// Archived emails
	db.Get(&overview.ArchivedEmails, `
		SELECT COUNT(*) FROM emails WHERE account_id = ? AND is_deleted = 0 AND is_archived = 1`,
		accountID)

	// Sent emails
	db.Get(&overview.SentEmails, `
		SELECT COUNT(*) FROM sent_emails WHERE account_id = ?`,
		accountID)

	// Draft count
	db.Get(&overview.DraftCount, `
		SELECT COUNT(*) FROM drafts WHERE account_id = ? AND status IN ('draft', 'scheduled')`,
		accountID)

	// Storage used (sum of email sizes in MB)
	db.Get(&overview.StorageUsedMB, `
		SELECT COALESCE(SUM(size) / 1048576.0, 0) FROM emails WHERE account_id = ? AND is_deleted = 0`,
		accountID)

	return &overview, nil
}

// SenderStatsResult contains statistics for a sender
type SenderStatsResult struct {
	Email       string `db:"from_email"`
	Name        string `db:"from_name"`
	Count       int    `db:"email_count"`
	UnreadCount int    `db:"unread_count"`
}

// GetTopSenders returns top email senders
func GetTopSenders(accountID int64, limit int, sinceDays int) ([]SenderStatsResult, error) {
	var senders []SenderStatsResult

	var query = `
		SELECT
			from_email,
			COALESCE(from_name, from_email) as from_name,
			COUNT(*) as email_count,
			SUM(CASE WHEN is_read = 0 THEN 1 ELSE 0 END) as unread_count
		FROM emails
		WHERE account_id = ? AND is_deleted = 0 AND is_archived = 0`

	if sinceDays > 0 {
		query += fmt.Sprintf(` AND date >= datetime('now', '-%d days')`, sinceDays)
	}

	query += `
		GROUP BY from_email
		ORDER BY email_count DESC
		LIMIT ?`

	err := db.Select(&senders, query, accountID, limit)
	return senders, err
}

// HourlyStatsResult contains email count per hour
type HourlyStatsResult struct {
	Hour  int `db:"hour"`
	Count int `db:"count"`
}

// GetEmailCountByHour returns email count by hour of day
func GetEmailCountByHour(accountID int64, sinceDays int) ([]HourlyStatsResult, error) {
	var stats []HourlyStatsResult

	var query = `
		SELECT
			CAST(strftime('%H', date) AS INTEGER) as hour,
			COUNT(*) as count
		FROM emails
		WHERE account_id = ? AND is_deleted = 0`

	if sinceDays > 0 {
		query += fmt.Sprintf(` AND date >= datetime('now', '-%d days')`, sinceDays)
	}

	query += `
		GROUP BY hour
		ORDER BY hour`

	err := db.Select(&stats, query, accountID)

	// Fill missing hours with 0
	var hourMap = make(map[int]int)
	for _, s := range stats {
		hourMap[s.Hour] = s.Count
	}

	var result = make([]HourlyStatsResult, 24)
	for i := 0; i < 24; i++ {
		result[i] = HourlyStatsResult{Hour: i, Count: hourMap[i]}
	}

	return result, err
}

// DailyStatsResult contains email count per day
type DailyStatsResult struct {
	Date  string `db:"date"`
	Count int    `db:"count"`
}

// GetEmailCountByDay returns email count by day
func GetEmailCountByDay(accountID int64, sinceDays int) ([]DailyStatsResult, error) {
	var stats []DailyStatsResult

	var query = `
		SELECT
			strftime('%Y-%m-%d', date) as date,
			COUNT(*) as count
		FROM emails
		WHERE account_id = ? AND is_deleted = 0`

	if sinceDays > 0 {
		query += fmt.Sprintf(` AND date >= datetime('now', '-%d days')`, sinceDays)
	}

	query += `
		GROUP BY date
		ORDER BY date DESC
		LIMIT ?`

	err := db.Select(&stats, query, accountID, sinceDays)
	return stats, err
}

// WeekdayStatsResult contains email count per weekday
type WeekdayStatsResult struct {
	Weekday int `db:"weekday"`
	Count   int `db:"count"`
}

// GetEmailCountByWeekday returns email count by day of week
func GetEmailCountByWeekday(accountID int64, sinceDays int) ([]WeekdayStatsResult, error) {
	var stats []WeekdayStatsResult

	var query = `
		SELECT
			CAST(strftime('%w', date) AS INTEGER) as weekday,
			COUNT(*) as count
		FROM emails
		WHERE account_id = ? AND is_deleted = 0`

	if sinceDays > 0 {
		query += fmt.Sprintf(` AND date >= datetime('now', '-%d days')`, sinceDays)
	}

	query += `
		GROUP BY weekday
		ORDER BY weekday`

	err := db.Select(&stats, query, accountID)

	// Fill missing weekdays with 0
	var weekdayMap = make(map[int]int)
	for _, s := range stats {
		weekdayMap[s.Weekday] = s.Count
	}

	var result = make([]WeekdayStatsResult, 7)
	for i := 0; i < 7; i++ {
		result[i] = WeekdayStatsResult{Weekday: i, Count: weekdayMap[i]}
	}

	return result, err
}

// ResponseStatsResult contains response time statistics
type ResponseStatsResult struct {
	AvgResponseMinutes float64 `db:"avg_response_minutes"`
	ResponseRate       float64 `db:"response_rate"`
}

// GetResponseStats returns response time statistics
func GetResponseStats(accountID int64) (*ResponseStatsResult, error) {
	var stats ResponseStatsResult

	// Calculate average response time for replied emails
	err := db.Get(&stats.AvgResponseMinutes, `
		SELECT COALESCE(AVG(
			(julianday(s.sent_at) - julianday(e.date)) * 24 * 60
		), 0)
		FROM sent_emails s
		JOIN emails e ON s.reply_to_email_id = e.id
		WHERE s.account_id = ? AND s.reply_to_email_id IS NOT NULL`,
		accountID)
	if err != nil {
		stats.AvgResponseMinutes = 0
	}

	// Calculate response rate (emails replied / total received)
	var total, replied int
	db.Get(&total, `SELECT COUNT(*) FROM emails WHERE account_id = ? AND is_deleted = 0`, accountID)
	db.Get(&replied, `SELECT COUNT(*) FROM emails WHERE account_id = ? AND is_deleted = 0 AND is_replied = 1`, accountID)

	if total > 0 {
		stats.ResponseRate = float64(replied) / float64(total) * 100
	}

	return &stats, nil
}

// === SYNC LOGS ===

// SyncLog representa um registro de sync
type SyncLog struct {
	ID            int64      `db:"id"`
	AccountID     int64      `db:"account_id"`
	FolderID      int64      `db:"folder_id"`
	StartedAt     time.Time  `db:"started_at"`
	CompletedAt   time.Time  `db:"completed_at"`
	NewEmails     int        `db:"new_emails"`
	DeletedEmails int        `db:"deleted_emails"`
	Error         sql.NullString `db:"error"`
}

// GetLastSyncTime retorna a data do último sync bem-sucedido
func GetLastSyncTime(accountID, folderID int64) (time.Time, error) {
	var completedAt time.Time
	err := db.Get(&completedAt, `
		SELECT completed_at FROM sync_logs
		WHERE account_id = ? AND folder_id = ? AND error IS NULL
		ORDER BY completed_at DESC LIMIT 1
	`, accountID, folderID)
	return completedAt, err
}

// CountNewEmailsSinceLastSync conta emails criados desde o último sync
func CountNewEmailsSinceLastSync(accountID, folderID int64) (int, error) {
	var lastSync, err = GetLastSyncTime(accountID, folderID)
	if err != nil {
		// Sem sync anterior, retorna 0
		return 0, nil
	}

	var count int
	err = db.Get(&count, `
		SELECT COUNT(*) FROM emails
		WHERE account_id = ? AND folder_id = ? AND is_deleted = 0
		AND created_at > ?
	`, accountID, folderID, lastSync)
	return count, err
}

// LogSyncStart registra o início de um sync e retorna o ID
func LogSyncStart(accountID, folderID int64) (int64, error) {
	// Limpa logs antigos (mais de 7 dias) para não crescer infinitamente
	db.Exec(`DELETE FROM sync_logs WHERE completed_at < datetime('now', '-7 days')`)

	var result, err = db.Exec(`
		INSERT INTO sync_logs (account_id, folder_id, started_at)
		VALUES (?, ?, ?)
	`, accountID, folderID, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// LogSyncComplete finaliza um sync com os resultados
func LogSyncComplete(syncID int64, newEmails, deletedEmails int, syncError error) error {
	var errStr sql.NullString
	if syncError != nil {
		errStr = sql.NullString{String: syncError.Error(), Valid: true}
	}

	_, err := db.Exec(`
		UPDATE sync_logs
		SET completed_at = ?, new_emails = ?, deleted_emails = ?, error = ?
		WHERE id = ?
	`, time.Now(), newEmails, deletedEmails, errStr, syncID)
	return err
}
