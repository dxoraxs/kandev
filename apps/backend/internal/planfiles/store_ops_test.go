package planfiles

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/db"
)

func TestStore_UpsertConfig_RoundTripsOperationSettings(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	cfg := sampleConfig("ws-1")
	cfg.ExecutorSteps = map[string]string{"step-x": "Claude"}
	cfg.NotesHeading = "Owner notes"
	cfg.WakeOnDate = false
	cfg.StaleAfterDays = 0
	cfg.IndexFile = "INDEX.md"

	saved, err := store.UpsertConfig(ctx, cfg)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"step-x": "Claude"}, saved.ExecutorSteps)
	assert.Equal(t, "Owner notes", saved.NotesHeading)
	assert.False(t, saved.WakeOnDate)
	assert.Equal(t, 0, saved.StaleAfterDays)
	assert.Equal(t, "INDEX.md", saved.IndexFile)
}

func TestStore_UpsertTaskRow_RoundTripsSyncedDependsOn(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	row := sampleTaskRow("task-1", "docs/plans/a.md")
	row.SyncedDependsOn = []string{"t-a", "t-b"}
	require.NoError(t, store.UpsertTaskRow(ctx, row))

	got, err := store.GetTaskRow(ctx, "task-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"t-a", "t-b"}, got.SyncedDependsOn)

	plain := sampleTaskRow("task-2", "docs/plans/b.md")
	require.NoError(t, store.UpsertTaskRow(ctx, plain))
	got, err = store.GetTaskRow(ctx, "task-2")
	require.NoError(t, err)
	assert.Empty(t, got.SyncedDependsOn)
}

const legacyConfigsTable = `CREATE TABLE plan_file_configs (
	workspace_id TEXT PRIMARY KEY,
	enabled INTEGER NOT NULL DEFAULT 0,
	workflow_id TEXT NOT NULL DEFAULT '',
	status_steps TEXT NOT NULL DEFAULT '{}',
	directories TEXT NOT NULL DEFAULT '[]',
	last_pass_at TIMESTAMP,
	last_pass_ok INTEGER NOT NULL DEFAULT 0,
	last_counts TEXT NOT NULL DEFAULT '{}',
	last_file_errors TEXT NOT NULL DEFAULT '[]',
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
)`

const legacyTasksTable = `CREATE TABLE plan_file_tasks (
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
	last_seen_at TIMESTAMP NOT NULL
)`

func TestStore_InitSchema_AddsColumnsToDatabaseCreatedBeforeThem(t *testing.T) {
	rawDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	rawDB.SetMaxOpenConns(1)
	conn := sqlx.NewDb(rawDB, "sqlite3")
	t.Cleanup(func() { _ = conn.Close() })
	for _, stmt := range []string{legacyConfigsTable, legacyTasksTable} {
		_, err := conn.Exec(stmt)
		require.NoError(t, err)
	}
	_, err = conn.Exec(`INSERT INTO plan_file_configs (workspace_id, enabled, workflow_id, created_at, updated_at)
		VALUES ('ws-old', 1, 'wf-1', '2026-01-01 00:00:00', '2026-01-01 00:00:00')`)
	require.NoError(t, err)
	_, err = conn.Exec(`INSERT INTO plan_file_tasks (task_id, workspace_id, repository_id, rel_path, external_id, last_seen_at)
		VALUES ('t-old', 'ws-old', 'r', 'a.md', 'plan-file:r:a.md', '2026-01-01 00:00:00')`)
	require.NoError(t, err)

	store, err := NewStore(conn, conn)
	require.NoError(t, err)
	_, err = NewStore(conn, conn)
	require.NoError(t, err, "a second init must replay cleanly")

	for _, c := range []string{"executor_steps", "notes_heading", "wake_on_date", "stale_after_days", "index_file"} {
		ok, err := db.ColumnExists(conn, "plan_file_configs", c)
		require.NoError(t, err)
		assert.True(t, ok, c)
	}
	ok, err := db.ColumnExists(conn, "plan_file_tasks", "synced_depends_on")
	require.NoError(t, err)
	assert.True(t, ok)

	cfg, err := store.GetConfig(context.Background(), "ws-old")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.ExecutorSteps)
	assert.Equal(t, "", cfg.NotesHeading)
	assert.True(t, cfg.WakeOnDate)
	assert.Equal(t, 7, cfg.StaleAfterDays)
	assert.Equal(t, "", cfg.IndexFile)
	row, err := store.GetTaskRow(context.Background(), "t-old")
	require.NoError(t, err)
	assert.Empty(t, row.SyncedDependsOn)
}
