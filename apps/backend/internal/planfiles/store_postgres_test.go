package planfiles

import (
	"context"
	"testing"
	"time"

	"github.com/kandev/kandev/internal/testutil"
)

func TestPostgresStoreSchemaReplay(t *testing.T) {
	ctx := context.Background()
	database := testutil.OpenIsolatedPostgres(t, testutil.PostgresDSNFromEnv(t))

	if _, err := NewStore(database, database); err != nil {
		t.Fatalf("first plan files store schema init: %v", err)
	}
	store, err := NewStore(database, database)
	if err != nil {
		t.Fatalf("second plan files store schema init: %v", err)
	}

	if _, err := store.UpsertConfig(ctx, sampleConfig("ws-1")); err != nil {
		t.Fatalf("upsert plan files config: %v", err)
	}
	at := time.Now().UTC().Truncate(time.Microsecond)
	if err := store.RecordPassStatus(ctx, "ws-1", at, true, PassCounts{Created: 1}, nil); err != nil {
		t.Fatalf("record pass status: %v", err)
	}
	cfg, err := store.GetConfig(ctx, "ws-1")
	if err != nil || cfg == nil || !cfg.LastPassOK || cfg.LastCounts.Created != 1 {
		t.Fatalf("plan files config = %+v, err=%v", cfg, err)
	}
	if err := store.UpsertTaskRow(ctx, sampleTaskRow("task-1", "docs/plans/a.md")); err != nil {
		t.Fatalf("upsert plan file task: %v", err)
	}
	rows, err := store.ListTaskRows(ctx, "ws-1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("plan file tasks = %+v, err=%v", rows, err)
	}
	if err := store.DeleteConfig(ctx, "ws-1"); err != nil {
		t.Fatalf("delete plan files config: %v", err)
	}
	if err := store.DeleteTaskRow(ctx, "task-1"); err != nil {
		t.Fatalf("delete plan file task: %v", err)
	}
}
