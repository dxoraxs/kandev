package worktree

import (
	"context"
	"testing"
)

func TestManagerGetAllByRepositoryIDReturnsOnlyThatRepository(t *testing.T) {
	cfg := newTestConfig(t)
	store := newMockStore()
	mgr, err := NewManager(cfg, store, newTestLogger())
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	store.worktrees["a"] = &Worktree{ID: "a", RepositoryID: "repo-1", Branch: "feature/a"}
	store.worktrees["b"] = &Worktree{ID: "b", RepositoryID: "repo-2", Branch: "feature/b"}

	got, err := mgr.GetAllByRepositoryID(context.Background(), "repo-1")
	if err != nil {
		t.Fatalf("GetAllByRepositoryID: %v", err)
	}
	if len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("worktrees = %+v, want only a", got)
	}
}
