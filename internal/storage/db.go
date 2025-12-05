package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT NOT NULL UNIQUE,
	name TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS folders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	total_messages INTEGER DEFAULT 0,
	unread_messages INTEGER DEFAULT 0,
	last_sync DATETIME,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	UNIQUE(account_id, name)
);

CREATE TABLE IF NOT EXISTS emails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	folder_id INTEGER NOT NULL,
	uid INTEGER NOT NULL,
	message_id TEXT,
	subject TEXT,
	from_name TEXT,
	from_email TEXT,
	to_addresses TEXT,
	cc_addresses TEXT,
	date DATETIME,
	is_read BOOLEAN DEFAULT 0,
	is_starred BOOLEAN DEFAULT 0,
	is_deleted BOOLEAN DEFAULT 0,
	has_attachments BOOLEAN DEFAULT 0,
	snippet TEXT,
	body_text TEXT,
	body_html TEXT,
	raw_headers TEXT,
	size INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	FOREIGN KEY (folder_id) REFERENCES folders(id),
	UNIQUE(account_id, folder_id, uid)
);

CREATE INDEX IF NOT EXISTS idx_emails_account_folder ON emails(account_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_emails_date ON emails(date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_from ON emails(from_email);
CREATE INDEX IF NOT EXISTS idx_emails_subject ON emails(subject);
CREATE INDEX IF NOT EXISTS idx_emails_is_read ON emails(is_read);

CREATE VIRTUAL TABLE IF NOT EXISTS emails_fts USING fts5(
	subject,
	from_name,
	from_email,
	body_text,
	content='emails',
	content_rowid='id',
	tokenize='trigram'
);

-- Triggers para manter FTS sincronizado
CREATE TRIGGER IF NOT EXISTS emails_ai AFTER INSERT ON emails BEGIN
	INSERT INTO emails_fts(rowid, subject, from_name, from_email, body_text)
	VALUES (new.id, new.subject, new.from_name, new.from_email, new.body_text);
END;

CREATE TRIGGER IF NOT EXISTS emails_ad AFTER DELETE ON emails BEGIN
	INSERT INTO emails_fts(emails_fts, rowid, subject, from_name, from_email, body_text)
	VALUES ('delete', old.id, old.subject, old.from_name, old.from_email, old.body_text);
END;

CREATE TRIGGER IF NOT EXISTS emails_au AFTER UPDATE ON emails BEGIN
	INSERT INTO emails_fts(emails_fts, rowid, subject, from_name, from_email, body_text)
	VALUES ('delete', old.id, old.subject, old.from_name, old.from_email, old.body_text);
	INSERT INTO emails_fts(rowid, subject, from_name, from_email, body_text)
	VALUES (new.id, new.subject, new.from_name, new.from_email, new.body_text);
END;

-- Tabela de drafts (rascunhos e emails agendados)
CREATE TABLE IF NOT EXISTS drafts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,

	-- Destinatários
	to_addresses TEXT NOT NULL,
	cc_addresses TEXT,
	bcc_addresses TEXT,

	-- Conteúdo
	subject TEXT NOT NULL,
	body_html TEXT,
	body_text TEXT,
	classification TEXT,

	-- Threading (se for reply)
	in_reply_to TEXT,
	reference_ids TEXT,
	reply_to_email_id INTEGER,

	-- Status e Timing
	status TEXT NOT NULL DEFAULT 'draft',
	scheduled_send_at DATETIME,
	sent_at DATETIME,

	-- Metadados
	generation_source TEXT NOT NULL DEFAULT 'manual',
	ai_prompt TEXT,
	error_message TEXT,

	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

	FOREIGN KEY (account_id) REFERENCES accounts(id),
	FOREIGN KEY (reply_to_email_id) REFERENCES emails(id)
);

CREATE INDEX IF NOT EXISTS idx_drafts_account_status ON drafts(account_id, status);
CREATE INDEX IF NOT EXISTS idx_drafts_scheduled ON drafts(status, scheduled_send_at);

-- Tabela de arquivo permanente de emails (nunca deletamos nada)
CREATE TABLE IF NOT EXISTS emails_archive (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	original_id INTEGER NOT NULL,
	account_id INTEGER NOT NULL,
	folder_id INTEGER NOT NULL,
	uid INTEGER NOT NULL,
	message_id TEXT,
	subject TEXT,
	from_name TEXT,
	from_email TEXT,
	to_addresses TEXT,
	cc_addresses TEXT,
	date DATETIME,
	is_read BOOLEAN DEFAULT 0,
	is_starred BOOLEAN DEFAULT 0,
	has_attachments BOOLEAN DEFAULT 0,
	snippet TEXT,
	body_text TEXT,
	body_html TEXT,
	raw_headers TEXT,
	size INTEGER DEFAULT 0,
	original_created_at DATETIME,
	original_updated_at DATETIME,
	archived_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	archive_reason TEXT NOT NULL, -- 'server_purged', 'user_deleted', 'manual_archive'
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_emails_archive_account ON emails_archive(account_id);
CREATE INDEX IF NOT EXISTS idx_emails_archive_date ON emails_archive(date DESC);
CREATE INDEX IF NOT EXISTS idx_emails_archive_from ON emails_archive(from_email);

-- Tabela de histórico de drafts (nunca deletamos nada)
CREATE TABLE IF NOT EXISTS drafts_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	original_id INTEGER NOT NULL,
	account_id INTEGER NOT NULL,
	to_addresses TEXT NOT NULL,
	cc_addresses TEXT,
	bcc_addresses TEXT,
	subject TEXT NOT NULL,
	body_html TEXT,
	body_text TEXT,
	classification TEXT,
	in_reply_to TEXT,
	reference_ids TEXT,
	reply_to_email_id INTEGER,
	final_status TEXT NOT NULL, -- 'sent', 'cancelled', 'deleted', 'failed'
	scheduled_send_at DATETIME,
	sent_at DATETIME,
	generation_source TEXT NOT NULL DEFAULT 'manual',
	ai_prompt TEXT,
	error_message TEXT,
	original_created_at DATETIME,
	original_updated_at DATETIME,
	archived_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_drafts_history_account ON drafts_history(account_id);
CREATE INDEX IF NOT EXISTS idx_drafts_history_status ON drafts_history(final_status);

-- Tabela de emails enviados (registro permanente)
CREATE TABLE IF NOT EXISTS sent_emails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	message_id TEXT,
	to_addresses TEXT NOT NULL,
	cc_addresses TEXT,
	bcc_addresses TEXT,
	subject TEXT NOT NULL,
	body_html TEXT,
	body_text TEXT,
	in_reply_to TEXT,
	reference_ids TEXT,
	reply_to_email_id INTEGER,
	sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	send_method TEXT NOT NULL, -- 'smtp', 'gmail_api'
	draft_id INTEGER, -- referência ao draft original se houver
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_sent_emails_account ON sent_emails(account_id);
CREATE INDEX IF NOT EXISTS idx_sent_emails_date ON sent_emails(sent_at DESC);

-- Tabela de operações em lote pendentes (preview antes de executar)
CREATE TABLE IF NOT EXISTS pending_batch_ops (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	operation TEXT NOT NULL, -- 'archive', 'delete', 'mark_read', 'mark_unread'
	description TEXT NOT NULL, -- descrição legível: "Arquivar 15 emails de newsletter@example.com"
	filter_query TEXT NOT NULL, -- query SQL ou descrição do filtro usado
	email_ids TEXT NOT NULL, -- IDs dos emails afetados (JSON array)
	email_count INTEGER NOT NULL,
	preview_data TEXT, -- JSON com preview dos emails (subject, from, date)
	status TEXT NOT NULL DEFAULT 'pending', -- pending, confirmed, cancelled, executed
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	executed_at DATETIME,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_pending_batch_ops_status ON pending_batch_ops(account_id, status);

-- Tabela de estado do indexador de conteúdo (background sync)
CREATE TABLE IF NOT EXISTS content_index_state (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL UNIQUE,
	status TEXT NOT NULL DEFAULT 'idle', -- idle, running, paused, completed, error
	total_emails INTEGER DEFAULT 0,
	indexed_emails INTEGER DEFAULT 0,
	last_indexed_uid INTEGER DEFAULT 0,
	speed INTEGER DEFAULT 100, -- emails por minuto
	last_error TEXT,
	started_at DATETIME,
	paused_at DATETIME,
	completed_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

-- Tabela de configurações do app
CREATE TABLE IF NOT EXISTS app_settings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	key TEXT NOT NULL,
	value TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(account_id, key)
);

CREATE INDEX IF NOT EXISTS idx_app_settings_account_key ON app_settings(account_id, key);

-- Tabela de logs de sync
CREATE TABLE IF NOT EXISTS sync_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	folder_id INTEGER NOT NULL,
	started_at DATETIME NOT NULL,
	completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	new_emails INTEGER DEFAULT 0,
	deleted_emails INTEGER DEFAULT 0,
	error TEXT,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	FOREIGN KEY (folder_id) REFERENCES folders(id)
);

CREATE INDEX IF NOT EXISTS idx_sync_logs_account_folder ON sync_logs(account_id, folder_id, completed_at DESC);

-- Tabela de metadados de anexos
CREATE TABLE IF NOT EXISTS attachments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email_id INTEGER NOT NULL,
	account_id INTEGER NOT NULL,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL,
	content_id TEXT,
	content_disposition TEXT,
	part_number TEXT,
	size INTEGER NOT NULL DEFAULT 0,
	checksum TEXT,
	encoding TEXT,
	charset TEXT,
	is_inline BOOLEAN DEFAULT 0,
	is_downloaded BOOLEAN DEFAULT 0,
	is_cached BOOLEAN DEFAULT 0,
	cache_path TEXT,
	cached_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	UNIQUE(email_id, filename)
);

CREATE INDEX IF NOT EXISTS idx_attachments_email ON attachments(email_id);
CREATE INDEX IF NOT EXISTS idx_attachments_account ON attachments(account_id);
CREATE INDEX IF NOT EXISTS idx_attachments_inline ON attachments(is_inline);

-- Tabela de cache de conteúdo binário de anexos
CREATE TABLE IF NOT EXISTS attachment_cache (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	attachment_id INTEGER NOT NULL UNIQUE,
	data BLOB NOT NULL,
	compressed BOOLEAN DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	last_accessed DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (attachment_id) REFERENCES attachments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_attachment_cache_last_accessed ON attachment_cache(last_accessed);

-- Tabela de histórico de operações (undo/redo)
CREATE TABLE IF NOT EXISTS operations_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	operation_type TEXT NOT NULL, -- 'mark_read', 'mark_starred', 'archive', 'delete', 'move', 'batch'
	operation_data TEXT NOT NULL, -- JSON com todos dados necessários para undo/redo
	description TEXT NOT NULL, -- descrição legível: "Arquivar email 'Assunto'"
	stack_type TEXT NOT NULL, -- 'undo' ou 'redo'
	stack_position INTEGER NOT NULL, -- posição na pilha (0 = mais recente)
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_operations_history_account_stack ON operations_history(account_id, stack_type, stack_position DESC);

-- Tabela de contatos (sincronizados do Google People API)
CREATE TABLE IF NOT EXISTS contacts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL,
	resource_name TEXT NOT NULL, -- people/c1234567890 (ID do Google)
	display_name TEXT,
	given_name TEXT,
	family_name TEXT,
	photo_url TEXT,
	photo_etag TEXT,
	photo_path TEXT, -- caminho local da foto cacheada
	is_starred BOOLEAN DEFAULT 0,
	interaction_count INTEGER DEFAULT 0, -- número de emails trocados
	last_interaction_at DATETIME,
	metadata_json TEXT, -- outros metadados do Google (JSON)
	synced_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	UNIQUE(account_id, resource_name)
);

CREATE INDEX IF NOT EXISTS idx_contacts_account ON contacts(account_id);
CREATE INDEX IF NOT EXISTS idx_contacts_display_name ON contacts(display_name);
CREATE INDEX IF NOT EXISTS idx_contacts_interaction ON contacts(account_id, interaction_count DESC);

-- Tabela de emails dos contatos (relação N:N, um contato pode ter múltiplos emails)
CREATE TABLE IF NOT EXISTS contact_emails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	contact_id INTEGER NOT NULL,
	email TEXT NOT NULL,
	email_type TEXT, -- home, work, home, other
	is_primary BOOLEAN DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE,
	UNIQUE(contact_id, email)
);

CREATE INDEX IF NOT EXISTS idx_contact_emails_contact ON contact_emails(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_emails_email ON contact_emails(email);

-- Tabela de telefones dos contatos
CREATE TABLE IF NOT EXISTS contact_phones (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	contact_id INTEGER NOT NULL,
	phone_number TEXT NOT NULL,
	phone_type TEXT, -- mobile, work, home, other
	is_primary BOOLEAN DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_contact_phones_contact ON contact_phones(contact_id);

-- Tabela de interações com contatos (histórico de emails)
CREATE TABLE IF NOT EXISTS contact_interactions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	contact_id INTEGER NOT NULL,
	email_id INTEGER, -- pode ser NULL se for email enviado que não está no DB
	interaction_type TEXT NOT NULL, -- 'received', 'sent'
	interaction_date DATETIME NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE,
	FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_contact_interactions_contact ON contact_interactions(contact_id, interaction_date DESC);
CREATE INDEX IF NOT EXISTS idx_contact_interactions_email ON contact_interactions(email_id);

-- Tabela de sync state para contatos (track last sync)
CREATE TABLE IF NOT EXISTS contacts_sync_state (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id INTEGER NOT NULL UNIQUE,
	last_sync_token TEXT,
	last_full_sync DATETIME,
	last_incremental_sync DATETIME,
	total_contacts INTEGER DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'never_synced', -- never_synced, syncing, synced, error
	error_message TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);
`

func Init(dbPath string) error {
	var dir = filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("erro ao criar diretório: %w", err)
	}

	var err error
	db, err = sqlx.Connect("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return fmt.Errorf("erro ao conectar ao banco: %w", err)
	}

	// Executa schema
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("erro ao criar schema: %w", err)
	}

	// Migração: atualiza FTS para trigram se necessário
	if err := migrateFTS(); err != nil {
		return fmt.Errorf("erro na migração FTS: %w", err)
	}

	// Migração: adiciona coluna is_replied
	if err := migrateAddIsReplied(); err != nil {
		return fmt.Errorf("erro na migração is_replied: %w", err)
	}

	// Migração: adiciona coluna is_archived
	if err := migrateAddIsArchived(); err != nil {
		return fmt.Errorf("erro na migração is_archived: %w", err)
	}

	// Migração: adiciona coluna body_indexed
	if err := migrateAddBodyIndexed(); err != nil {
		return fmt.Errorf("erro na migração body_indexed: %w", err)
	}

	// Migração: adiciona colunas de threading
	if err := migrateAddThreading(); err != nil {
		return fmt.Errorf("erro na migração threading: %w", err)
	}

	// Migração: adiciona coluna forward_to para batch ops
	if err := migrateAddForwardTo(); err != nil {
		return fmt.Errorf("erro na migração forward_to: %w", err)
	}

	// Migração: tabelas de plugins
	if err := InitPluginTables(); err != nil {
		return fmt.Errorf("erro na migração plugins: %w", err)
	}

	return nil
}

// migrateAddIsReplied adiciona coluna is_replied se não existir
func migrateAddIsReplied() error {
	var _, err = db.Exec("ALTER TABLE emails ADD COLUMN is_replied BOOLEAN DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	return nil
}

// migrateAddIsArchived adiciona coluna is_archived se não existir
func migrateAddIsArchived() error {
	var _, err = db.Exec("ALTER TABLE emails ADD COLUMN is_archived BOOLEAN DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	// Adiciona índice para is_archived
	db.Exec("CREATE INDEX IF NOT EXISTS idx_emails_is_archived ON emails(is_archived)")
	return nil
}

// migrateAddBodyIndexed adiciona coluna body_indexed para tracking do indexador
func migrateAddBodyIndexed() error {
	var _, err = db.Exec("ALTER TABLE emails ADD COLUMN body_indexed BOOLEAN DEFAULT 0")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	// Adiciona índice para body_indexed
	db.Exec("CREATE INDEX IF NOT EXISTS idx_emails_body_indexed ON emails(body_indexed)")
	return nil
}

// migrateAddThreading adiciona colunas de threading (in_reply_to, references, thread_id)
func migrateAddThreading() error {
	// Adiciona in_reply_to (Message-ID do email sendo respondido)
	var _, err = db.Exec("ALTER TABLE emails ADD COLUMN in_reply_to TEXT")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}

	// Adiciona references (lista completa de Message-IDs da thread)
	// "references" is a SQLite reserved keyword, must be quoted
	_, err = db.Exec(`ALTER TABLE emails ADD COLUMN "references" TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}

	// Adiciona thread_id (identificador normalizado da thread)
	_, err = db.Exec("ALTER TABLE emails ADD COLUMN thread_id TEXT")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}

	// Cria índices para queries rápidas de threads
	db.Exec("CREATE INDEX IF NOT EXISTS idx_emails_thread_id ON emails(thread_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_emails_message_id ON emails(message_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_emails_in_reply_to ON emails(in_reply_to)")

	return nil
}

// migrateAddForwardTo adiciona coluna forward_to para operações de forward em batch
func migrateAddForwardTo() error {
	var _, err = db.Exec("ALTER TABLE pending_batch_ops ADD COLUMN forward_to TEXT")
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return err
	}
	return nil
}

// migrateFTS verifica e recria FTS com trigram tokenizer
func migrateFTS() error {
	// Verifica se FTS usa trigram
	var sql string
	err := db.Get(&sql, "SELECT sql FROM sqlite_master WHERE type='table' AND name='emails_fts'")
	if err != nil {
		return nil // tabela não existe ainda
	}

	// Se já tem trigram, não precisa migrar
	if strings.Contains(sql, "trigram") {
		return nil
	}

	// Recria FTS com trigram
	var _, err2 = db.Exec(`
		DROP TABLE IF EXISTS emails_fts;
		CREATE VIRTUAL TABLE emails_fts USING fts5(
			subject,
			from_name,
			from_email,
			body_text,
			content='emails',
			content_rowid='id',
			tokenize='trigram'
		);
		INSERT INTO emails_fts(rowid, subject, from_name, from_email, body_text)
		SELECT id, subject, from_name, from_email, body_text FROM emails;
	`)
	return err2
}

func GetDB() *sqlx.DB {
	return db
}

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
