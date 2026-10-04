package planfiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/kandev/kandev/internal/db/dialect"
	"github.com/kandev/kandev/internal/planfiles/format"
)

// Store owns plan_file_configs and plan_file_tasks.
type Store struct {
	db *sqlx.DB
	ro *sqlx.DB
}

// NewStore creates a Store and initializes the schema if needed.
func NewStore(writer, reader *sqlx.DB) (*Store, error) {
	s := &Store{db: writer, ro: reader}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("planfiles schema init: %w", err)
	}
	return s, nil
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS plan_file_configs (
		workspace_id TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 0,
		workflow_id TEXT NOT NULL DEFAULT '',
		status_steps TEXT NOT NULL DEFAULT '{}',
		directories TEXT NOT NULL DEFAULT '[]',
		last_pass_at {{timestamp}},
		last_pass_ok INTEGER NOT NULL DEFAULT 0,
		last_counts TEXT NOT NULL DEFAULT '{}',
		last_file_errors TEXT NOT NULL DEFAULT '[]',
		created_at {{timestamp}} NOT NULL,
		updated_at {{timestamp}} NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS plan_file_tasks (
		task_id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		repository_id TEXT NOT NULL,
		rel_path TEXT NOT NULL,
		external_id TEXT NOT NULL,
		content_hash TEXT NOT NULL DEFAULT '',
		synced_step_id TEXT NOT NULL DEFAULT '',
		synced_priority TEXT NOT NULL DEFAULT '',
		synced_order_key TEXT NOT NULL DEFAULT '',
		notice TEXT NOT NULL DEFAULT '',
		last_seen_at {{timestamp}} NOT NULL
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_file_tasks_path
		ON plan_file_tasks (workspace_id, repository_id, rel_path)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_file_tasks_external_id
		ON plan_file_tasks (workspace_id, external_id)`,
	`CREATE INDEX IF NOT EXISTS idx_plan_file_tasks_workspace
		ON plan_file_tasks (workspace_id)`,
}

func (s *Store) initSchema() error {
	for _, stmt := range schemaStatements {
		if _, err := s.db.Exec(dialect.MustRenderSchema(s.db.DriverName(), stmt)); err != nil {
			return err
		}
	}
	return nil
}

const configColumns = `workspace_id, enabled, workflow_id, status_steps, directories, last_pass_at,
	last_pass_ok, last_counts, last_file_errors, created_at, updated_at`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanConfig(row rowScanner) (*Config, error) {
	cfg := &Config{}
	var enabled, lastOK int
	var lastPassAt sql.NullTime
	var statusSteps, directories, counts, fileErrors string
	if err := row.Scan(&cfg.WorkspaceID, &enabled, &cfg.WorkflowID, &statusSteps, &directories,
		&lastPassAt, &lastOK, &counts, &fileErrors, &cfg.CreatedAt, &cfg.UpdatedAt); err != nil {
		return nil, err
	}
	cfg.Enabled = enabled != 0
	cfg.LastPassOK = lastOK != 0
	if lastPassAt.Valid {
		t := lastPassAt.Time
		cfg.LastPassAt = &t
	}
	// A corrupt JSON column degrades to its empty value rather than failing
	// the read, so the owner can still open and repair the settings.
	_ = json.Unmarshal([]byte(statusSteps), &cfg.StatusSteps)
	_ = json.Unmarshal([]byte(directories), &cfg.Directories)
	_ = json.Unmarshal([]byte(counts), &cfg.LastCounts)
	_ = json.Unmarshal([]byte(fileErrors), &cfg.LastFileErrors)
	if cfg.StatusSteps == nil {
		cfg.StatusSteps = map[format.BoardStatus]string{}
	}
	return cfg, nil
}

// GetConfig returns the workspace's config, or (nil, nil) when none is stored.
func (s *Store) GetConfig(ctx context.Context, workspaceID string) (*Config, error) {
	row := s.ro.QueryRowContext(ctx, s.ro.Rebind(
		`SELECT `+configColumns+` FROM plan_file_configs WHERE workspace_id = ?`), workspaceID)
	cfg, err := scanConfig(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// ListEnabledConfigs returns every config whose sync is switched on, ordered
// by workspace ID.
func (s *Store) ListEnabledConfigs(ctx context.Context) ([]*Config, error) {
	rows, err := s.ro.QueryContext(ctx, s.ro.Rebind(
		`SELECT `+configColumns+` FROM plan_file_configs WHERE enabled = 1 ORDER BY workspace_id`))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []*Config
	for rows.Next() {
		cfg, err := scanConfig(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, cfg)
	}
	return result, rows.Err()
}

// UpsertConfig creates or replaces the settings of a workspace's config
// (enabled, workflow, mapping, directories). The last-pass status columns are
// left untouched.
func (s *Store) UpsertConfig(ctx context.Context, cfg *Config) (*Config, error) {
	statusSteps, err := json.Marshal(cfg.StatusSteps)
	if err != nil {
		return nil, err
	}
	directories, err := json.Marshal(cfg.Directories)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, s.db.Rebind(`
		INSERT INTO plan_file_configs (workspace_id, enabled, workflow_id, status_steps, directories, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id) DO UPDATE SET
			enabled = excluded.enabled,
			workflow_id = excluded.workflow_id,
			status_steps = excluded.status_steps,
			directories = excluded.directories,
			updated_at = excluded.updated_at
	`), cfg.WorkspaceID, boolToInt(cfg.Enabled), cfg.WorkflowID, string(statusSteps), string(directories), now, now)
	if err != nil {
		return nil, err
	}
	return s.getConfigFromWriter(ctx, cfg.WorkspaceID)
}

// getConfigFromWriter reads back through the writer so a read replica that
// lags behind the write cannot return the previous row.
func (s *Store) getConfigFromWriter(ctx context.Context, workspaceID string) (*Config, error) {
	row := s.db.QueryRowContext(ctx, s.db.Rebind(
		`SELECT `+configColumns+` FROM plan_file_configs WHERE workspace_id = ?`), workspaceID)
	return scanConfig(row)
}

// RecordPassStatus stores the outcome of a sync pass. File errors beyond
// MaxStoredFileErrors are dropped.
func (s *Store) RecordPassStatus(
	ctx context.Context, workspaceID string, at time.Time, ok bool, counts PassCounts, fileErrors []FileErrorRow,
) error {
	if len(fileErrors) > MaxStoredFileErrors {
		fileErrors = fileErrors[:MaxStoredFileErrors]
	}
	if fileErrors == nil {
		fileErrors = []FileErrorRow{}
	}
	countsJSON, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	errorsJSON, err := json.Marshal(fileErrors)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.db.Rebind(`
		UPDATE plan_file_configs
		SET last_pass_at = ?, last_pass_ok = ?, last_counts = ?, last_file_errors = ?, updated_at = ?
		WHERE workspace_id = ?
	`), at, boolToInt(ok), string(countsJSON), string(errorsJSON), at, workspaceID)
	return err
}

// DeleteConfig removes a workspace's config. Deleting a missing config is a
// no-op.
func (s *Store) DeleteConfig(ctx context.Context, workspaceID string) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind(`DELETE FROM plan_file_configs WHERE workspace_id = ?`), workspaceID)
	return err
}

const taskColumns = `task_id, workspace_id, repository_id, rel_path, external_id, content_hash,
	synced_step_id, synced_priority, synced_order_key, notice, last_seen_at`

func scanTaskRow(row rowScanner) (*TaskRow, error) {
	r := &TaskRow{}
	if err := row.Scan(&r.TaskID, &r.WorkspaceID, &r.RepositoryID, &r.RelPath, &r.ExternalID, &r.ContentHash,
		&r.SyncedStepID, &r.SyncedPriority, &r.SyncedOrderKey, &r.Notice, &r.LastSeenAt); err != nil {
		return nil, err
	}
	return r, nil
}

// GetTaskRow returns the sync row of a task, or (nil, nil) when the task is
// not a plan task.
func (s *Store) GetTaskRow(ctx context.Context, taskID string) (*TaskRow, error) {
	row := s.ro.QueryRowContext(ctx, s.ro.Rebind(
		`SELECT `+taskColumns+` FROM plan_file_tasks WHERE task_id = ?`), taskID)
	r, err := scanTaskRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// ListTaskRows returns every plan task row of a workspace ordered by task ID.
func (s *Store) ListTaskRows(ctx context.Context, workspaceID string) ([]*TaskRow, error) {
	rows, err := s.ro.QueryContext(ctx, s.ro.Rebind(
		`SELECT `+taskColumns+` FROM plan_file_tasks WHERE workspace_id = ? ORDER BY task_id`), workspaceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []*TaskRow
	for rows.Next() {
		r, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// UpsertTaskRow inserts or replaces the row of a task. The unique indexes on
// (workspace, repository, path) and (workspace, external id) reject a second
// row for the same file or identifier.
func (s *Store) UpsertTaskRow(ctx context.Context, r *TaskRow) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind(`
		INSERT INTO plan_file_tasks (`+taskColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(task_id) DO UPDATE SET
			workspace_id = excluded.workspace_id,
			repository_id = excluded.repository_id,
			rel_path = excluded.rel_path,
			external_id = excluded.external_id,
			content_hash = excluded.content_hash,
			synced_step_id = excluded.synced_step_id,
			synced_priority = excluded.synced_priority,
			synced_order_key = excluded.synced_order_key,
			notice = excluded.notice,
			last_seen_at = excluded.last_seen_at
	`), r.TaskID, r.WorkspaceID, r.RepositoryID, r.RelPath, r.ExternalID, r.ContentHash,
		r.SyncedStepID, r.SyncedPriority, r.SyncedOrderKey, r.Notice, r.LastSeenAt)
	return err
}

// DeleteTaskRow removes the row of a task. Deleting a missing row is a no-op.
func (s *Store) DeleteTaskRow(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind(`DELETE FROM plan_file_tasks WHERE task_id = ?`), taskID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
