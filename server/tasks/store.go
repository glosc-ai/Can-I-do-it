package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gloscai/template-go-vue3-docker/server/database"
)

var ErrNotFound = errors.New("task not found")

type SQLStore struct {
	db     *sql.DB
	driver string
}

func NewSQLStore(db *sql.DB, driver string) *SQLStore {
	return &SQLStore{db: db, driver: driver}
}

func (s *SQLStore) List(ctx context.Context, userID int64) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		database.Placeholder(s.driver, `
		SELECT id, title, completed, created_at
		FROM tasks
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 100`), userID)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		items = append(items, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}
	return items, nil
}

func (s *SQLStore) Create(ctx context.Context, userID int64, title string) (Task, error) {
	if s.driver == "postgres" {
		var task Task
		err := s.db.QueryRowContext(ctx, `
			INSERT INTO tasks (user_id, title)
			VALUES ($1, $2)
			RETURNING id, title, completed, created_at`, userID, title).
			Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
		if err != nil {
			return Task{}, fmt.Errorf("creating task: %w", err)
		}
		return task, nil
	}

	result, err := s.db.ExecContext(ctx, "INSERT INTO tasks (user_id, title) VALUES (?, ?)", userID, title)
	if err != nil {
		return Task{}, fmt.Errorf("creating task: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("reading created task ID: %w", err)
	}
	return s.lookup(ctx, id)
}

// lookup fetches a task by ID without any user ownership check — it is only
// called internally after a write that already validates ownership.
func (s *SQLStore) lookup(ctx context.Context, id int64) (Task, error) {
	query := database.Placeholder(s.driver, "SELECT id, title, completed, created_at FROM tasks WHERE id = ?")
	var task Task
	if err := s.db.QueryRowContext(ctx, query, id).Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("looking up task %d: %w", id, err)
	}
	return task, nil
}

func (s *SQLStore) SetCompleted(ctx context.Context, userID, id int64, completed bool) (Task, error) {
	query := database.Placeholder(s.driver, "UPDATE tasks SET completed = ? WHERE id = ? AND user_id = ?")
	result, err := s.db.ExecContext(ctx, query, completed, id, userID)
	if err != nil {
		return Task{}, fmt.Errorf("updating task %d: %w", id, err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return Task{}, fmt.Errorf("reading updated row count: %w", err)
	}
	if updated == 0 {
		return Task{}, ErrNotFound
	}
	return s.lookup(ctx, id)
}

func (s *SQLStore) Delete(ctx context.Context, userID, id int64) error {
	query := database.Placeholder(s.driver, "DELETE FROM tasks WHERE id = ? AND user_id = ?")
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("deleting task %d: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading deleted row count: %w", err)
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}
