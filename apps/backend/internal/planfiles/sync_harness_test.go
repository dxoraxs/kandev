package planfiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

const (
	testRepoID   = "repo-1"
	testRepoName = "city_companion"
	plansDir     = "docs/plans"
	handoffStep  = "extra"
)

// syncHarness wires a plan-file service to the in-memory task system and a
// real temporary repository directory.
type syncHarness struct {
	t     *testing.T
	svc   *Service
	tasks *fakeTaskSystem
	wf    *fakeWorkflows
	root  string
}

func newSyncHarness(t *testing.T) *syncHarness {
	t.Helper()
	svc, wf := newTestService(t)
	tasks := newFakeTaskSystem()
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	tasks.repos = []*taskmodels.Repository{{
		ID: testRepoID, WorkspaceID: testWorkspace, Name: testRepoName, SourceType: "local", LocalPath: root,
	}}
	svc.SetSyncDeps(tasks, tasks)
	_, err = svc.PutConfig(context.Background(), testWorkspace, validRequest())
	require.NoError(t, err)
	return &syncHarness{t: t, svc: svc, tasks: tasks, wf: wf, root: root}
}

// step returns the board step a status is mapped to in the test config.
func (h *syncHarness) step(status format.BoardStatus) string {
	return validRequest().StatusSteps[status]
}

func (h *syncHarness) writeFile(rel, content string) {
	h.t.Helper()
	full := filepath.Join(h.root, filepath.FromSlash(rel))
	require.NoError(h.t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(h.t, os.WriteFile(full, []byte(content), 0o644))
}

func (h *syncHarness) removeFile(rel string) {
	h.t.Helper()
	require.NoError(h.t, os.Remove(filepath.Join(h.root, filepath.FromSlash(rel))))
}

// planDoc builds a plan file. extra holds additional frontmatter lines.
func planDoc(board, heading string, extra ...string) string {
	var b strings.Builder
	b.WriteString("---\nboard: " + board + "\n")
	for _, line := range extra {
		b.WriteString(line + "\n")
	}
	b.WriteString("---\n\n# " + heading + "\n\nBody of " + heading + ".\n")
	return b.String()
}

func (h *syncHarness) sync() PassSummary {
	h.t.Helper()
	summary, err := h.svc.SyncWorkspace(context.Background(), testWorkspace)
	require.NoError(h.t, err)
	return summary
}

func (h *syncHarness) taskFor(rel string) *taskmodels.Task {
	h.t.Helper()
	task := h.tasks.byExternalID(defaultExternalID(testRepoID, rel))
	require.NotNil(h.t, task, "no task for %s", rel)
	return task
}

func noMoveOptions() taskservice.MoveTaskOptions {
	return taskservice.MoveTaskOptions{}
}
