package planfiles

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	rawDB, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	// Each new connection to an in-memory SQLite database is isolated, so the
	// pool is pinned to one connection.
	rawDB.SetMaxOpenConns(1)
	db := sqlx.NewDb(rawDB, "sqlite3")
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db, db)
	require.NoError(t, err)
	return store
}

func sampleConfig(workspaceID string) *Config {
	return &Config{
		WorkspaceID: workspaceID,
		Enabled:     true,
		WorkflowID:  "wf-1",
		StatusSteps: map[format.BoardStatus]string{
			format.BoardQueued: "step-q",
			format.BoardDone:   "step-d",
		},
		Directories: []string{"docs/plans"},
	}
}

func sampleTaskRow(taskID, relPath string) *TaskRow {
	return &TaskRow{
		TaskID:         taskID,
		WorkspaceID:    "ws-1",
		RepositoryID:   "repo-1",
		RelPath:        relPath,
		ExternalID:     "plan-file:repo-1:" + relPath,
		ContentHash:    "hash-1",
		SyncedStepID:   "step-q",
		SyncedPriority: "medium",
		SyncedOrderKey: "1:10:" + relPath,
		LastSeenAt:     time.Now().UTC().Truncate(time.Second),
	}
}

func TestStore_GetConfig_MissingReturnsNil(t *testing.T) {
	store := setupTestStore(t)
	cfg, err := store.GetConfig(context.Background(), "ws-1")
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestStore_UpsertConfig_RoundTrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	saved, err := store.UpsertConfig(ctx, sampleConfig("ws-1"))
	require.NoError(t, err)
	assert.True(t, saved.Enabled)
	assert.Equal(t, "wf-1", saved.WorkflowID)
	assert.Equal(t, "step-q", saved.StatusSteps[format.BoardQueued])
	assert.Equal(t, []string{"docs/plans"}, saved.Directories)
	assert.Nil(t, saved.LastPassAt)
	assert.False(t, saved.CreatedAt.IsZero())
}

func TestStore_UpsertConfig_ReplacesSettingsAndKeepsPassStatus(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	_, err := store.UpsertConfig(ctx, sampleConfig("ws-1"))
	require.NoError(t, err)
	at := time.Now().UTC().Truncate(time.Second)
	counts := PassCounts{Created: 2, Failed: 1}
	rows := []FileErrorRow{{RepositoryID: "repo-1", RepositoryName: "city", RelPath: "docs/plans/a.md", Reason: "board value"}}
	require.NoError(t, store.RecordPassStatus(ctx, "ws-1", at, false, counts, rows))

	next := sampleConfig("ws-1")
	next.Enabled = false
	next.Directories = []string{"plans", "docs/superpowers/plans"}
	saved, err := store.UpsertConfig(ctx, next)
	require.NoError(t, err)

	assert.False(t, saved.Enabled)
	assert.Equal(t, []string{"plans", "docs/superpowers/plans"}, saved.Directories)
	require.NotNil(t, saved.LastPassAt, "saving settings must not erase the last pass status")
	assert.False(t, saved.LastPassOK)
	assert.Equal(t, counts, saved.LastCounts)
	assert.Equal(t, rows, saved.LastFileErrors)
}

func TestStore_RecordPassStatus_CapsFileErrors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	_, err := store.UpsertConfig(ctx, sampleConfig("ws-1"))
	require.NoError(t, err)

	rows := make([]FileErrorRow, MaxStoredFileErrors+25)
	for i := range rows {
		rows[i] = FileErrorRow{RepositoryID: "repo-1", RelPath: "docs/plans/x.md", Reason: "parse"}
	}
	require.NoError(t, store.RecordPassStatus(ctx, "ws-1", time.Now().UTC(), true, PassCounts{}, rows))

	cfg, err := store.GetConfig(ctx, "ws-1")
	require.NoError(t, err)
	assert.Len(t, cfg.LastFileErrors, MaxStoredFileErrors)
	assert.True(t, cfg.LastPassOK)
}

func TestStore_DeleteConfig(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	_, err := store.UpsertConfig(ctx, sampleConfig("ws-1"))
	require.NoError(t, err)

	require.NoError(t, store.DeleteConfig(ctx, "ws-1"))
	cfg, err := store.GetConfig(ctx, "ws-1")
	require.NoError(t, err)
	assert.Nil(t, cfg)
	require.NoError(t, store.DeleteConfig(ctx, "ws-1"), "deleting a missing config is a no-op")
}

func TestStore_TaskRow_UpsertGetAndUpdate(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	missing, err := store.GetTaskRow(ctx, "task-1")
	require.NoError(t, err)
	assert.Nil(t, missing)

	row := sampleTaskRow("task-1", "docs/plans/a.md")
	require.NoError(t, store.UpsertTaskRow(ctx, row))
	got, err := store.GetTaskRow(ctx, "task-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, row.ExternalID, got.ExternalID)
	assert.Equal(t, "1:10:docs/plans/a.md", got.SyncedOrderKey)
	assert.Empty(t, got.Notice)

	row.ContentHash = "hash-2"
	row.Notice = "board edit not saved"
	require.NoError(t, store.UpsertTaskRow(ctx, row))
	got, err = store.GetTaskRow(ctx, "task-1")
	require.NoError(t, err)
	assert.Equal(t, "hash-2", got.ContentHash)
	assert.Equal(t, "board edit not saved", got.Notice)
}

func TestStore_ListTaskRows_ScopedToWorkspace(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.UpsertTaskRow(ctx, sampleTaskRow("task-1", "docs/plans/a.md")))
	require.NoError(t, store.UpsertTaskRow(ctx, sampleTaskRow("task-2", "docs/plans/b.md")))
	other := sampleTaskRow("task-3", "docs/plans/a.md")
	other.WorkspaceID = "ws-2"
	other.ExternalID = "plan-file:repo-1:docs/plans/a.md:ws-2"
	require.NoError(t, store.UpsertTaskRow(ctx, other))

	rows, err := store.ListTaskRows(ctx, "ws-1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "task-1", rows[0].TaskID)
	assert.Equal(t, "task-2", rows[1].TaskID)
}

func TestStore_TaskRow_UniquePathAndExternalID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.UpsertTaskRow(ctx, sampleTaskRow("task-1", "docs/plans/a.md")))

	samePath := sampleTaskRow("task-2", "docs/plans/a.md")
	samePath.ExternalID = "different"
	assert.Error(t, store.UpsertTaskRow(ctx, samePath), "one row per (workspace, repository, path)")

	sameExternal := sampleTaskRow("task-3", "docs/plans/c.md")
	sameExternal.ExternalID = "plan-file:repo-1:docs/plans/a.md"
	assert.Error(t, store.UpsertTaskRow(ctx, sameExternal), "one row per (workspace, external id)")
}

func TestStore_DeleteTaskRow(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.UpsertTaskRow(ctx, sampleTaskRow("task-1", "docs/plans/a.md")))

	require.NoError(t, store.DeleteTaskRow(ctx, "task-1"))
	got, err := store.GetTaskRow(ctx, "task-1")
	require.NoError(t, err)
	assert.Nil(t, got)
	require.NoError(t, store.DeleteTaskRow(ctx, "task-1"), "deleting a missing row is a no-op")
}

func TestStore_SchemaInitIsIdempotent(t *testing.T) {
	store := setupTestStore(t)
	_, err := NewStore(store.db, store.ro)
	require.NoError(t, err)
}

func TestStore_ListEnabledConfigs_ReturnsOnlyEnabledOrderedByWorkspace(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	for _, ws := range []string{"ws-b", "ws-a", "ws-off"} {
		cfg := sampleConfig(ws)
		cfg.Enabled = ws != "ws-off"
		_, err := store.UpsertConfig(ctx, cfg)
		require.NoError(t, err)
	}

	got, err := store.ListEnabledConfigs(ctx)

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "ws-a", got[0].WorkspaceID)
	assert.Equal(t, "ws-b", got[1].WorkspaceID)
}
