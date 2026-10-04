package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/kandev/kandev/internal/task/models"
)

// The external plan synchronizer writes one running position across every
// column, so the `position` sort must follow it regardless of recency, with
// the title as the tie-breaker.
func TestQuerySidebarTaskPageSortsByBoardPosition(t *testing.T) {
	repo := newRepoForEntityTests(t)
	ctx := context.Background()
	seedWorkspace(t, repo, "ws-sidebar-position")
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	rows := []struct {
		id, title string
		position  int
	}{
		{"p-third", "Alpha", 2},
		{"p-first", "Zulu", 0},
		{"p-tie-b", "Beta", 1},
		{"p-tie-a", "Alpha tie", 1},
	}
	for index, row := range rows {
		// Later rows are newer, so a recency sort would invert the expectation.
		if err := repo.CreateTask(ctx, &models.Task{
			ID: row.id, WorkspaceID: "ws-sidebar-position", Title: row.title, Position: row.position,
			CreatedAt: base.Add(time.Duration(index) * time.Minute),
			UpdatedAt: base.Add(time.Duration(index) * time.Minute),
		}); err != nil {
			t.Fatalf("create %s: %v", row.id, err)
		}
	}
	query := sidebarTaskQuery(1)
	query.Sort = models.SidebarTaskViewSort{Key: "position", Direction: "asc"}
	page, err := repo.QuerySidebarTaskPage(ctx, "ws-sidebar-position", query, models.SidebarTaskViewPreferences{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	got := sidebarTaskIDs(page.Tasks)
	want := []string{"p-first", "p-tie-a", "p-tie-b", "p-third"}
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}
