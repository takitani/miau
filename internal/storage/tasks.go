package storage

import (
	"database/sql"
	"time"
)

// === TASKS ===

// CreateTask cria uma nova tarefa
func CreateTask(task *Task) error {
	var result, err = db.Exec(`
		INSERT INTO tasks (account_id, title, description, is_completed, priority, due_date, email_id, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.AccountID, task.Title, task.Description, task.IsCompleted,
		task.Priority, task.DueDate, task.EmailID, task.Source)
	if err != nil {
		return err
	}

	var id, _ = result.LastInsertId()
	task.ID = id
	task.CreatedAt = SQLiteTime{time.Now()}
	task.UpdatedAt = SQLiteTime{time.Now()}
	return nil
}

// GetTask retorna uma tarefa por ID
func GetTask(id int64) (*Task, error) {
	var task Task
	err := db.Get(&task, "SELECT * FROM tasks WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &task, nil
}

// GetTasks retorna todas as tarefas de uma conta
func GetTasks(accountID int64) ([]Task, error) {
	var tasks []Task
	err := db.Select(&tasks, `
		SELECT * FROM tasks
		WHERE account_id = ?
		ORDER BY is_completed ASC, priority DESC, created_at DESC`,
		accountID)
	return tasks, err
}

// GetPendingTasks retorna apenas tarefas não completadas
func GetPendingTasks(accountID int64) ([]Task, error) {
	var tasks []Task
	err := db.Select(&tasks, `
		SELECT * FROM tasks
		WHERE account_id = ? AND is_completed = 0
		ORDER BY priority DESC, due_date ASC NULLS LAST, created_at DESC`,
		accountID)
	return tasks, err
}

// GetCompletedTasks retorna apenas tarefas completadas
func GetCompletedTasks(accountID int64, limit int) ([]Task, error) {
	var tasks []Task
	err := db.Select(&tasks, `
		SELECT * FROM tasks
		WHERE account_id = ? AND is_completed = 1
		ORDER BY updated_at DESC
		LIMIT ?`,
		accountID, limit)
	return tasks, err
}

// GetTasksByEmail retorna tarefas associadas a um email
func GetTasksByEmail(emailID int64) ([]Task, error) {
	var tasks []Task
	err := db.Select(&tasks, `
		SELECT * FROM tasks
		WHERE email_id = ?
		ORDER BY created_at DESC`,
		emailID)
	return tasks, err
}

// GetTasksBySource retorna tarefas por origem (manual ou AI)
func GetTasksBySource(accountID int64, source TaskSource) ([]Task, error) {
	var tasks []Task
	err := db.Select(&tasks, `
		SELECT * FROM tasks
		WHERE account_id = ? AND source = ?
		ORDER BY is_completed ASC, priority DESC, created_at DESC`,
		accountID, source)
	return tasks, err
}

// UpdateTask atualiza uma tarefa existente
func UpdateTask(task *Task) error {
	_, err := db.Exec(`
		UPDATE tasks SET
			title = ?,
			description = ?,
			is_completed = ?,
			priority = ?,
			due_date = ?,
			email_id = ?,
			source = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		task.Title, task.Description, task.IsCompleted,
		task.Priority, task.DueDate, task.EmailID, task.Source,
		task.ID)
	return err
}

// ToggleTaskCompleted alterna o status de completado
func ToggleTaskCompleted(id int64) (bool, error) {
	var task Task
	err := db.Get(&task, "SELECT is_completed FROM tasks WHERE id = ?", id)
	if err != nil {
		return false, err
	}

	var newStatus = !task.IsCompleted
	_, err = db.Exec(`
		UPDATE tasks SET is_completed = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		newStatus, id)
	return newStatus, err
}

// DeleteTask remove uma tarefa
func DeleteTask(id int64) error {
	_, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

// DeleteCompletedTasks remove todas as tarefas completadas de uma conta
func DeleteCompletedTasks(accountID int64) (int64, error) {
	result, err := db.Exec(`
		DELETE FROM tasks
		WHERE account_id = ? AND is_completed = 1`,
		accountID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CountTasks retorna contagem de tarefas por status
func CountTasks(accountID int64) (pending, completed int, err error) {
	err = db.Get(&pending, `
		SELECT COUNT(*) FROM tasks
		WHERE account_id = ? AND is_completed = 0`,
		accountID)
	if err != nil {
		return
	}

	err = db.Get(&completed, `
		SELECT COUNT(*) FROM tasks
		WHERE account_id = ? AND is_completed = 1`,
		accountID)
	return
}
