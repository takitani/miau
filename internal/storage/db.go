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
