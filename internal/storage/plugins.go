// Package storage implements plugin data persistence.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opik/miau/internal/ports"
)

// pluginSchema defines the plugin-related tables
const pluginSchema = `
-- Plugin state (enabled/connected status per account)
CREATE TABLE IF NOT EXISTS plugin_states (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plugin_id TEXT NOT NULL,
	account_id INTEGER NOT NULL,
	status TEXT NOT NULL DEFAULT 'disabled',
	error TEXT,
	last_sync_at DATETIME,
	item_count INTEGER DEFAULT 0,
	external_id TEXT,
	external_name TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	UNIQUE(plugin_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_states_account ON plugin_states(account_id);
CREATE INDEX IF NOT EXISTS idx_plugin_states_plugin ON plugin_states(plugin_id);

-- Plugin credentials (encrypted tokens, API keys)
CREATE TABLE IF NOT EXISTS plugin_credentials (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plugin_id TEXT NOT NULL,
	account_id INTEGER NOT NULL,
	credentials_json TEXT NOT NULL, -- JSON map of credentials
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	UNIQUE(plugin_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_credentials_account ON plugin_credentials(account_id);

-- External projects from plugins
CREATE TABLE IF NOT EXISTS external_projects (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plugin_id TEXT NOT NULL,
	account_id INTEGER NOT NULL,
	external_id TEXT NOT NULL, -- ID in external system
	name TEXT NOT NULL,
	description TEXT,
	url TEXT,
	status TEXT DEFAULT 'active',
	color TEXT,
	icon TEXT,
	creator_id TEXT,
	creator_name TEXT,
	item_count INTEGER DEFAULT 0,
	metadata_json TEXT,
	created_at DATETIME,
	updated_at DATETIME,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	UNIQUE(plugin_id, account_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_external_projects_account ON external_projects(account_id, plugin_id);
CREATE INDEX IF NOT EXISTS idx_external_projects_name ON external_projects(name);

-- External items (tasks, messages, documents, etc.)
CREATE TABLE IF NOT EXISTS external_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plugin_id TEXT NOT NULL,
	account_id INTEGER NOT NULL,
	external_id TEXT NOT NULL, -- ID in external system
	project_id TEXT,
	project_name TEXT,
	item_type TEXT NOT NULL, -- task, message, comment, document, event
	title TEXT,
	content TEXT,
	content_html TEXT,
	url TEXT,
	status TEXT,
	priority TEXT,
	due_at DATETIME,
	created_at DATETIME,
	updated_at DATETIME,
	completed_at DATETIME,
	creator_id TEXT,
	creator_name TEXT,
	creator_email TEXT,
	assignees_json TEXT, -- JSON array of assignees
	tags_json TEXT, -- JSON array of tags
	attachments_json TEXT, -- JSON array of attachments
	parent_id TEXT,
	comment_count INTEGER DEFAULT 0,
	metadata_json TEXT,
	synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX IF NOT EXISTS idx_external_items_account ON external_items(account_id, plugin_id);
CREATE INDEX IF NOT EXISTS idx_external_items_project ON external_items(plugin_id, project_id);
CREATE INDEX IF NOT EXISTS idx_external_items_type ON external_items(item_type);
CREATE INDEX IF NOT EXISTS idx_external_items_status ON external_items(status);
CREATE INDEX IF NOT EXISTS idx_external_items_due ON external_items(due_at);
CREATE INDEX IF NOT EXISTS idx_external_items_updated ON external_items(updated_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_external_items_unique ON external_items(plugin_id, account_id, external_id);

-- FTS for external items
CREATE VIRTUAL TABLE IF NOT EXISTS external_items_fts USING fts5(
	title,
	content,
	project_name,
	creator_name,
	content='external_items',
	content_rowid='id',
	tokenize='trigram'
);

-- FTS triggers
CREATE TRIGGER IF NOT EXISTS external_items_ai AFTER INSERT ON external_items BEGIN
	INSERT INTO external_items_fts(rowid, title, content, project_name, creator_name)
	VALUES (new.id, new.title, new.content, new.project_name, new.creator_name);
END;

CREATE TRIGGER IF NOT EXISTS external_items_ad AFTER DELETE ON external_items BEGIN
	INSERT INTO external_items_fts(external_items_fts, rowid, title, content, project_name, creator_name)
	VALUES ('delete', old.id, old.title, old.content, old.project_name, old.creator_name);
END;

CREATE TRIGGER IF NOT EXISTS external_items_au AFTER UPDATE ON external_items BEGIN
	INSERT INTO external_items_fts(external_items_fts, rowid, title, content, project_name, creator_name)
	VALUES ('delete', old.id, old.title, old.content, old.project_name, old.creator_name);
	INSERT INTO external_items_fts(rowid, title, content, project_name, creator_name)
	VALUES (new.id, new.title, new.content, new.project_name, new.creator_name);
END;
`

// InitPluginTables creates plugin-related tables
func InitPluginTables() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := db.Exec(pluginSchema)
	return err
}

// PluginStorage implements ports.PluginStoragePort
type PluginStorage struct{}

// NewPluginStorage creates a new plugin storage instance
func NewPluginStorage() *PluginStorage {
	return &PluginStorage{}
}

// SavePluginState saves or updates plugin state
func (s *PluginStorage) SavePluginState(ctx context.Context, state *ports.PluginState) error {
	query := `
		INSERT INTO plugin_states (plugin_id, account_id, status, error, last_sync_at, item_count, external_id, external_name, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(plugin_id, account_id) DO UPDATE SET
			status = excluded.status,
			error = excluded.error,
			last_sync_at = excluded.last_sync_at,
			item_count = excluded.item_count,
			external_id = excluded.external_id,
			external_name = excluded.external_name,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.ExecContext(ctx, query,
		state.PluginID, state.AccountID, state.Status, state.Error,
		state.LastSyncAt, state.ItemCount, state.ExternalID, state.ExternalName)
	return err
}

// GetPluginState retrieves plugin state
func (s *PluginStorage) GetPluginState(ctx context.Context, pluginID ports.PluginID, accountID int64) (*ports.PluginState, error) {
	var state struct {
		PluginID     string         `db:"plugin_id"`
		AccountID    int64          `db:"account_id"`
		Status       string         `db:"status"`
		Error        sql.NullString `db:"error"`
		LastSyncAt   sql.NullTime   `db:"last_sync_at"`
		ItemCount    int            `db:"item_count"`
		ExternalID   sql.NullString `db:"external_id"`
		ExternalName sql.NullString `db:"external_name"`
	}

	query := `SELECT plugin_id, account_id, status, error, last_sync_at, item_count, external_id, external_name
		FROM plugin_states WHERE plugin_id = ? AND account_id = ?`
	err := db.GetContext(ctx, &state, query, pluginID, accountID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result := &ports.PluginState{
		PluginID:     ports.PluginID(state.PluginID),
		AccountID:    state.AccountID,
		Status:       ports.PluginStatus(state.Status),
		ItemCount:    state.ItemCount,
		ExternalID:   state.ExternalID.String,
		ExternalName: state.ExternalName.String,
	}
	if state.Error.Valid {
		result.Error = state.Error.String
	}
	if state.LastSyncAt.Valid {
		result.LastSyncAt = &state.LastSyncAt.Time
	}
	return result, nil
}

// GetAllPluginStates retrieves all plugin states for an account
func (s *PluginStorage) GetAllPluginStates(ctx context.Context, accountID int64) ([]ports.PluginState, error) {
	query := `SELECT plugin_id, account_id, status, error, last_sync_at, item_count, external_id, external_name
		FROM plugin_states WHERE account_id = ?`
	rows, err := db.QueryxContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []ports.PluginState
	for rows.Next() {
		var state struct {
			PluginID     string         `db:"plugin_id"`
			AccountID    int64          `db:"account_id"`
			Status       string         `db:"status"`
			Error        sql.NullString `db:"error"`
			LastSyncAt   sql.NullTime   `db:"last_sync_at"`
			ItemCount    int            `db:"item_count"`
			ExternalID   sql.NullString `db:"external_id"`
			ExternalName sql.NullString `db:"external_name"`
		}
		if err := rows.StructScan(&state); err != nil {
			return nil, err
		}
		result := ports.PluginState{
			PluginID:     ports.PluginID(state.PluginID),
			AccountID:    state.AccountID,
			Status:       ports.PluginStatus(state.Status),
			ItemCount:    state.ItemCount,
			ExternalID:   state.ExternalID.String,
			ExternalName: state.ExternalName.String,
		}
		if state.Error.Valid {
			result.Error = state.Error.String
		}
		if state.LastSyncAt.Valid {
			result.LastSyncAt = &state.LastSyncAt.Time
		}
		states = append(states, result)
	}
	return states, nil
}

// DeletePluginState removes plugin state
func (s *PluginStorage) DeletePluginState(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	_, err := db.ExecContext(ctx, "DELETE FROM plugin_states WHERE plugin_id = ? AND account_id = ?", pluginID, accountID)
	return err
}

// SavePluginCredentials saves plugin credentials
func (s *PluginStorage) SavePluginCredentials(ctx context.Context, pluginID ports.PluginID, accountID int64, creds map[string]string) error {
	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO plugin_credentials (plugin_id, account_id, credentials_json, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(plugin_id, account_id) DO UPDATE SET
			credentials_json = excluded.credentials_json,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = db.ExecContext(ctx, query, pluginID, accountID, string(credsJSON))
	return err
}

// GetPluginCredentials retrieves plugin credentials
func (s *PluginStorage) GetPluginCredentials(ctx context.Context, pluginID ports.PluginID, accountID int64) (map[string]string, error) {
	var credsJSON string
	query := `SELECT credentials_json FROM plugin_credentials WHERE plugin_id = ? AND account_id = ?`
	err := db.GetContext(ctx, &credsJSON, query, pluginID, accountID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var creds map[string]string
	if err := json.Unmarshal([]byte(credsJSON), &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

// DeletePluginCredentials removes plugin credentials
func (s *PluginStorage) DeletePluginCredentials(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	_, err := db.ExecContext(ctx, "DELETE FROM plugin_credentials WHERE plugin_id = ? AND account_id = ?", pluginID, accountID)
	return err
}

// SaveExternalProjects saves external projects
func (s *PluginStorage) SaveExternalProjects(ctx context.Context, pluginID ports.PluginID, accountID int64, projects []ports.ExternalProject) error {
	query := `
		INSERT INTO external_projects (
			plugin_id, account_id, external_id, name, description, url, status, color, icon,
			creator_id, creator_name, item_count, metadata_json, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(plugin_id, account_id, external_id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			url = excluded.url,
			status = excluded.status,
			color = excluded.color,
			icon = excluded.icon,
			creator_id = excluded.creator_id,
			creator_name = excluded.creator_name,
			item_count = excluded.item_count,
			metadata_json = excluded.metadata_json,
			updated_at = excluded.updated_at,
			synced_at = CURRENT_TIMESTAMP
	`

	for _, p := range projects {
		var metadataJSON []byte
		if p.Metadata != nil {
			metadataJSON, _ = json.Marshal(p.Metadata)
		}
		var creatorID, creatorName string
		if p.Creator != nil {
			creatorID = p.Creator.ID
			creatorName = p.Creator.Name
		}

		_, err := db.ExecContext(ctx, query,
			pluginID, accountID, p.ID, p.Name, p.Description, p.URL, p.Status,
			p.Color, p.Icon, creatorID, creatorName, p.ItemCount,
			string(metadataJSON), p.CreatedAt, p.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetExternalProjects retrieves external projects
func (s *PluginStorage) GetExternalProjects(ctx context.Context, pluginID ports.PluginID, accountID int64) ([]ports.ExternalProject, error) {
	query := `
		SELECT external_id, name, description, url, status, color, icon,
			creator_id, creator_name, item_count, metadata_json, created_at, updated_at
		FROM external_projects
		WHERE plugin_id = ? AND account_id = ?
		ORDER BY name
	`
	rows, err := db.QueryxContext(ctx, query, pluginID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []ports.ExternalProject
	for rows.Next() {
		var p struct {
			ExternalID   string         `db:"external_id"`
			Name         string         `db:"name"`
			Description  sql.NullString `db:"description"`
			URL          sql.NullString `db:"url"`
			Status       sql.NullString `db:"status"`
			Color        sql.NullString `db:"color"`
			Icon         sql.NullString `db:"icon"`
			CreatorID    sql.NullString `db:"creator_id"`
			CreatorName  sql.NullString `db:"creator_name"`
			ItemCount    int            `db:"item_count"`
			MetadataJSON sql.NullString `db:"metadata_json"`
			CreatedAt    sql.NullTime   `db:"created_at"`
			UpdatedAt    sql.NullTime   `db:"updated_at"`
		}
		if err := rows.StructScan(&p); err != nil {
			return nil, err
		}

		project := ports.ExternalProject{
			ID:          p.ExternalID,
			PluginID:    pluginID,
			Name:        p.Name,
			Description: p.Description.String,
			URL:         p.URL.String,
			Status:      p.Status.String,
			Color:       p.Color.String,
			Icon:        p.Icon.String,
			ItemCount:   p.ItemCount,
		}
		if p.CreatorID.Valid {
			project.Creator = &ports.ExternalPerson{
				ID:   p.CreatorID.String,
				Name: p.CreatorName.String,
			}
		}
		if p.MetadataJSON.Valid {
			json.Unmarshal([]byte(p.MetadataJSON.String), &project.Metadata)
		}
		if p.CreatedAt.Valid {
			project.CreatedAt = p.CreatedAt.Time
		}
		if p.UpdatedAt.Valid {
			project.UpdatedAt = p.UpdatedAt.Time
		}
		projects = append(projects, project)
	}
	return projects, nil
}

// SaveExternalItems saves external items
func (s *PluginStorage) SaveExternalItems(ctx context.Context, pluginID ports.PluginID, accountID int64, items []ports.ExternalItem) error {
	query := `
		INSERT INTO external_items (
			plugin_id, account_id, external_id, project_id, project_name, item_type,
			title, content, content_html, url, status, priority, due_at,
			created_at, updated_at, completed_at, creator_id, creator_name, creator_email,
			assignees_json, tags_json, attachments_json, parent_id, comment_count, metadata_json, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(plugin_id, account_id, external_id) DO UPDATE SET
			project_id = excluded.project_id,
			project_name = excluded.project_name,
			title = excluded.title,
			content = excluded.content,
			content_html = excluded.content_html,
			url = excluded.url,
			status = excluded.status,
			priority = excluded.priority,
			due_at = excluded.due_at,
			updated_at = excluded.updated_at,
			completed_at = excluded.completed_at,
			assignees_json = excluded.assignees_json,
			tags_json = excluded.tags_json,
			attachments_json = excluded.attachments_json,
			comment_count = excluded.comment_count,
			metadata_json = excluded.metadata_json,
			synced_at = CURRENT_TIMESTAMP
	`

	for _, item := range items {
		var creatorID, creatorName, creatorEmail string
		if item.Creator != nil {
			creatorID = item.Creator.ID
			creatorName = item.Creator.Name
			creatorEmail = item.Creator.Email
		}

		assigneesJSON, _ := json.Marshal(item.Assignees)
		tagsJSON, _ := json.Marshal(item.Tags)
		attachmentsJSON, _ := json.Marshal(item.Attachments)
		metadataJSON, _ := json.Marshal(item.Metadata)

		_, err := db.ExecContext(ctx, query,
			pluginID, accountID, item.ID, item.ProjectID, item.ProjectName, item.Type,
			item.Title, item.Content, item.ContentHTML, item.URL, item.Status, item.Priority, item.DueAt,
			item.CreatedAt, item.UpdatedAt, item.CompletedAt, creatorID, creatorName, creatorEmail,
			string(assigneesJSON), string(tagsJSON), string(attachmentsJSON),
			item.ParentID, item.CommentCount, string(metadataJSON))
		if err != nil {
			return err
		}
	}
	return nil
}

// GetExternalItems retrieves external items with query options
func (s *PluginStorage) GetExternalItems(ctx context.Context, pluginID ports.PluginID, accountID int64, opts ports.ExternalItemQuery) ([]ports.ExternalItem, error) {
	query := `
		SELECT external_id, project_id, project_name, item_type, title, content, content_html,
			url, status, priority, due_at, created_at, updated_at, completed_at,
			creator_id, creator_name, creator_email, assignees_json, tags_json,
			attachments_json, parent_id, comment_count, metadata_json
		FROM external_items
		WHERE plugin_id = ? AND account_id = ?
	`
	args := []interface{}{pluginID, accountID}

	if opts.ProjectID != "" {
		query += " AND project_id = ?"
		args = append(args, opts.ProjectID)
	}
	if len(opts.Types) > 0 {
		query += " AND item_type IN ("
		for i, t := range opts.Types {
			if i > 0 {
				query += ","
			}
			query += "?"
			args = append(args, t)
		}
		query += ")"
	}
	if opts.Status == "pending" {
		query += " AND status != 'completed'"
	} else if opts.Status == "completed" {
		query += " AND status = 'completed'"
	}
	if opts.Since != nil {
		query += " AND updated_at >= ?"
		args = append(args, opts.Since)
	}

	query += " ORDER BY updated_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanExternalItems(rows, pluginID)
}

// GetExternalItem retrieves a single external item
func (s *PluginStorage) GetExternalItem(ctx context.Context, pluginID ports.PluginID, itemID string) (*ports.ExternalItem, error) {
	query := `
		SELECT external_id, project_id, project_name, item_type, title, content, content_html,
			url, status, priority, due_at, created_at, updated_at, completed_at,
			creator_id, creator_name, creator_email, assignees_json, tags_json,
			attachments_json, parent_id, comment_count, metadata_json
		FROM external_items
		WHERE plugin_id = ? AND external_id = ?
	`
	rows, err := db.QueryxContext(ctx, query, pluginID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items, err := s.scanExternalItems(rows, pluginID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

// DeleteExternalItems removes specific external items
func (s *PluginStorage) DeleteExternalItems(ctx context.Context, pluginID ports.PluginID, accountID int64, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return nil
	}

	query := "DELETE FROM external_items WHERE plugin_id = ? AND account_id = ? AND external_id IN ("
	args := []interface{}{pluginID, accountID}
	for i, id := range itemIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// DeleteAllPluginItems removes all items for a plugin
func (s *PluginStorage) DeleteAllPluginItems(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	_, err := db.ExecContext(ctx, "DELETE FROM external_items WHERE plugin_id = ? AND account_id = ?", pluginID, accountID)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "DELETE FROM external_projects WHERE plugin_id = ? AND account_id = ?", pluginID, accountID)
	return err
}

// scanExternalItems scans rows into ExternalItem slice
func (s *PluginStorage) scanExternalItems(rows *sql.Rows, pluginID ports.PluginID) ([]ports.ExternalItem, error) {
	var items []ports.ExternalItem

	for rows.Next() {
		var item struct {
			ExternalID      string         `db:"external_id"`
			ProjectID       sql.NullString `db:"project_id"`
			ProjectName     sql.NullString `db:"project_name"`
			ItemType        string         `db:"item_type"`
			Title           sql.NullString `db:"title"`
			Content         sql.NullString `db:"content"`
			ContentHTML     sql.NullString `db:"content_html"`
			URL             sql.NullString `db:"url"`
			Status          sql.NullString `db:"status"`
			Priority        sql.NullString `db:"priority"`
			DueAt           sql.NullTime   `db:"due_at"`
			CreatedAt       sql.NullTime   `db:"created_at"`
			UpdatedAt       sql.NullTime   `db:"updated_at"`
			CompletedAt     sql.NullTime   `db:"completed_at"`
			CreatorID       sql.NullString `db:"creator_id"`
			CreatorName     sql.NullString `db:"creator_name"`
			CreatorEmail    sql.NullString `db:"creator_email"`
			AssigneesJSON   sql.NullString `db:"assignees_json"`
			TagsJSON        sql.NullString `db:"tags_json"`
			AttachmentsJSON sql.NullString `db:"attachments_json"`
			ParentID        sql.NullString `db:"parent_id"`
			CommentCount    int            `db:"comment_count"`
			MetadataJSON    sql.NullString `db:"metadata_json"`
		}

		if err := rows.Scan(
			&item.ExternalID, &item.ProjectID, &item.ProjectName, &item.ItemType,
			&item.Title, &item.Content, &item.ContentHTML, &item.URL,
			&item.Status, &item.Priority, &item.DueAt, &item.CreatedAt,
			&item.UpdatedAt, &item.CompletedAt, &item.CreatorID, &item.CreatorName,
			&item.CreatorEmail, &item.AssigneesJSON, &item.TagsJSON,
			&item.AttachmentsJSON, &item.ParentID, &item.CommentCount, &item.MetadataJSON,
		); err != nil {
			return nil, err
		}

		result := ports.ExternalItem{
			ID:           item.ExternalID,
			PluginID:     pluginID,
			ProjectID:    item.ProjectID.String,
			ProjectName:  item.ProjectName.String,
			Type:         ports.ExternalItemType(item.ItemType),
			Title:        item.Title.String,
			Content:      item.Content.String,
			ContentHTML:  item.ContentHTML.String,
			URL:          item.URL.String,
			Status:       item.Status.String,
			Priority:     item.Priority.String,
			ParentID:     item.ParentID.String,
			CommentCount: item.CommentCount,
		}

		if item.DueAt.Valid {
			result.DueAt = &item.DueAt.Time
		}
		if item.CreatedAt.Valid {
			result.CreatedAt = item.CreatedAt.Time
		}
		if item.UpdatedAt.Valid {
			result.UpdatedAt = item.UpdatedAt.Time
		}
		if item.CompletedAt.Valid {
			result.CompletedAt = &item.CompletedAt.Time
		}
		if item.CreatorID.Valid {
			result.Creator = &ports.ExternalPerson{
				ID:    item.CreatorID.String,
				Name:  item.CreatorName.String,
				Email: item.CreatorEmail.String,
			}
		}
		if item.AssigneesJSON.Valid {
			json.Unmarshal([]byte(item.AssigneesJSON.String), &result.Assignees)
		}
		if item.TagsJSON.Valid {
			json.Unmarshal([]byte(item.TagsJSON.String), &result.Tags)
		}
		if item.AttachmentsJSON.Valid {
			json.Unmarshal([]byte(item.AttachmentsJSON.String), &result.Attachments)
		}
		if item.MetadataJSON.Valid {
			json.Unmarshal([]byte(item.MetadataJSON.String), &result.Metadata)
		}

		items = append(items, result)
	}

	return items, nil
}

// SearchExternalItems performs full-text search on external items
func (s *PluginStorage) SearchExternalItems(ctx context.Context, accountID int64, query string, limit int) ([]ports.ExternalItem, error) {
	sqlQuery := `
		SELECT e.external_id, e.plugin_id, e.project_id, e.project_name, e.item_type,
			e.title, e.content, e.content_html, e.url, e.status, e.priority, e.due_at,
			e.created_at, e.updated_at, e.completed_at, e.creator_id, e.creator_name,
			e.creator_email, e.assignees_json, e.tags_json, e.attachments_json,
			e.parent_id, e.comment_count, e.metadata_json
		FROM external_items e
		JOIN external_items_fts fts ON e.id = fts.rowid
		WHERE e.account_id = ? AND external_items_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`
	rows, err := db.QueryxContext(ctx, sqlQuery, accountID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ports.ExternalItem
	for rows.Next() {
		var item struct {
			ExternalID      string         `db:"external_id"`
			PluginID        string         `db:"plugin_id"`
			ProjectID       sql.NullString `db:"project_id"`
			ProjectName     sql.NullString `db:"project_name"`
			ItemType        string         `db:"item_type"`
			Title           sql.NullString `db:"title"`
			Content         sql.NullString `db:"content"`
			ContentHTML     sql.NullString `db:"content_html"`
			URL             sql.NullString `db:"url"`
			Status          sql.NullString `db:"status"`
			Priority        sql.NullString `db:"priority"`
			DueAt           sql.NullTime   `db:"due_at"`
			CreatedAt       sql.NullTime   `db:"created_at"`
			UpdatedAt       sql.NullTime   `db:"updated_at"`
			CompletedAt     sql.NullTime   `db:"completed_at"`
			CreatorID       sql.NullString `db:"creator_id"`
			CreatorName     sql.NullString `db:"creator_name"`
			CreatorEmail    sql.NullString `db:"creator_email"`
			AssigneesJSON   sql.NullString `db:"assignees_json"`
			TagsJSON        sql.NullString `db:"tags_json"`
			AttachmentsJSON sql.NullString `db:"attachments_json"`
			ParentID        sql.NullString `db:"parent_id"`
			CommentCount    int            `db:"comment_count"`
			MetadataJSON    sql.NullString `db:"metadata_json"`
		}
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}

		result := ports.ExternalItem{
			ID:           item.ExternalID,
			PluginID:     ports.PluginID(item.PluginID),
			ProjectID:    item.ProjectID.String,
			ProjectName:  item.ProjectName.String,
			Type:         ports.ExternalItemType(item.ItemType),
			Title:        item.Title.String,
			Content:      item.Content.String,
			ContentHTML:  item.ContentHTML.String,
			URL:          item.URL.String,
			Status:       item.Status.String,
			Priority:     item.Priority.String,
			ParentID:     item.ParentID.String,
			CommentCount: item.CommentCount,
		}
		if item.DueAt.Valid {
			result.DueAt = &item.DueAt.Time
		}
		if item.CreatedAt.Valid {
			result.CreatedAt = item.CreatedAt.Time
		}
		if item.UpdatedAt.Valid {
			result.UpdatedAt = item.UpdatedAt.Time
		}
		if item.CompletedAt.Valid {
			result.CompletedAt = &item.CompletedAt.Time
		}
		if item.CreatorID.Valid {
			result.Creator = &ports.ExternalPerson{
				ID:    item.CreatorID.String,
				Name:  item.CreatorName.String,
				Email: item.CreatorEmail.String,
			}
		}
		if item.AssigneesJSON.Valid {
			json.Unmarshal([]byte(item.AssigneesJSON.String), &result.Assignees)
		}
		if item.TagsJSON.Valid {
			json.Unmarshal([]byte(item.TagsJSON.String), &result.Tags)
		}
		if item.AttachmentsJSON.Valid {
			json.Unmarshal([]byte(item.AttachmentsJSON.String), &result.Attachments)
		}
		if item.MetadataJSON.Valid {
			json.Unmarshal([]byte(item.MetadataJSON.String), &result.Metadata)
		}

		items = append(items, result)
	}
	return items, nil
}

// GetRecentItems returns recently updated items across all plugins
func (s *PluginStorage) GetRecentItems(ctx context.Context, accountID int64, limit int, since *time.Time) ([]ports.ExternalItem, error) {
	query := `
		SELECT external_id, plugin_id, project_id, project_name, item_type,
			title, content, content_html, url, status, priority, due_at,
			created_at, updated_at, completed_at, creator_id, creator_name,
			creator_email, assignees_json, tags_json, attachments_json,
			parent_id, comment_count, metadata_json
		FROM external_items
		WHERE account_id = ?
	`
	args := []interface{}{accountID}

	if since != nil {
		query += " AND updated_at >= ?"
		args = append(args, since)
	}

	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ports.ExternalItem
	for rows.Next() {
		var item struct {
			ExternalID      string         `db:"external_id"`
			PluginID        string         `db:"plugin_id"`
			ProjectID       sql.NullString `db:"project_id"`
			ProjectName     sql.NullString `db:"project_name"`
			ItemType        string         `db:"item_type"`
			Title           sql.NullString `db:"title"`
			Content         sql.NullString `db:"content"`
			ContentHTML     sql.NullString `db:"content_html"`
			URL             sql.NullString `db:"url"`
			Status          sql.NullString `db:"status"`
			Priority        sql.NullString `db:"priority"`
			DueAt           sql.NullTime   `db:"due_at"`
			CreatedAt       sql.NullTime   `db:"created_at"`
			UpdatedAt       sql.NullTime   `db:"updated_at"`
			CompletedAt     sql.NullTime   `db:"completed_at"`
			CreatorID       sql.NullString `db:"creator_id"`
			CreatorName     sql.NullString `db:"creator_name"`
			CreatorEmail    sql.NullString `db:"creator_email"`
			AssigneesJSON   sql.NullString `db:"assignees_json"`
			TagsJSON        sql.NullString `db:"tags_json"`
			AttachmentsJSON sql.NullString `db:"attachments_json"`
			ParentID        sql.NullString `db:"parent_id"`
			CommentCount    int            `db:"comment_count"`
			MetadataJSON    sql.NullString `db:"metadata_json"`
		}
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}

		result := ports.ExternalItem{
			ID:           item.ExternalID,
			PluginID:     ports.PluginID(item.PluginID),
			ProjectID:    item.ProjectID.String,
			ProjectName:  item.ProjectName.String,
			Type:         ports.ExternalItemType(item.ItemType),
			Title:        item.Title.String,
			Content:      item.Content.String,
			ContentHTML:  item.ContentHTML.String,
			URL:          item.URL.String,
			Status:       item.Status.String,
			Priority:     item.Priority.String,
			ParentID:     item.ParentID.String,
			CommentCount: item.CommentCount,
		}
		if item.DueAt.Valid {
			result.DueAt = &item.DueAt.Time
		}
		if item.CreatedAt.Valid {
			result.CreatedAt = item.CreatedAt.Time
		}
		if item.UpdatedAt.Valid {
			result.UpdatedAt = item.UpdatedAt.Time
		}
		if item.CompletedAt.Valid {
			result.CompletedAt = &item.CompletedAt.Time
		}
		if item.CreatorID.Valid {
			result.Creator = &ports.ExternalPerson{
				ID:    item.CreatorID.String,
				Name:  item.CreatorName.String,
				Email: item.CreatorEmail.String,
			}
		}
		if item.AssigneesJSON.Valid {
			json.Unmarshal([]byte(item.AssigneesJSON.String), &result.Assignees)
		}
		if item.TagsJSON.Valid {
			json.Unmarshal([]byte(item.TagsJSON.String), &result.Tags)
		}
		if item.AttachmentsJSON.Valid {
			json.Unmarshal([]byte(item.AttachmentsJSON.String), &result.Attachments)
		}
		if item.MetadataJSON.Valid {
			json.Unmarshal([]byte(item.MetadataJSON.String), &result.Metadata)
		}

		items = append(items, result)
	}
	return items, nil
}
