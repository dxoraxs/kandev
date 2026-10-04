package planfiles

import (
	"context"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
)

// counter reads one outcome of plan_files_writeback_total.
func writebackCount(outcome string) int64 {
	if v, ok := writebackTotal.Get("outcome=" + outcome).(interface{ Value() int64 }); ok {
		return v.Value()
	}
	return 0
}

func (h *syncHarness) writeBack() *WriteBackSubscriber {
	return NewWriteBackSubscriber(h.svc, nil, logger.Default())
}

func (h *syncHarness) read(rel string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.root, filepath.FromSlash(rel)))
	require.NoError(h.t, err)
	return string(data)
}

// mtime returns the modification time of a plan file in nanoseconds.
func (h *syncHarness) mtime(rel string) int64 {
	h.t.Helper()
	info, err := os.Stat(filepath.Join(h.root, filepath.FromSlash(rel)))
	require.NoError(h.t, err)
	return info.ModTime().UnixNano()
}

func (h *syncHarness) row(rel string) *TaskRow {
	h.t.Helper()
	row, err := h.svc.store.GetTaskRow(context.Background(), h.taskFor(rel).ID)
	require.NoError(h.t, err)
	require.NotNil(h.t, row)
	return row
}

// boardMove is a person dragging a plan task to another step.
func (h *syncHarness) boardMove(rel, stepID string) {
	h.t.Helper()
	_, err := h.tasks.MoveTaskWithOptions(context.Background(), h.taskFor(rel).ID, "wf-1", stepID, 0, noMoveOptions())
	require.NoError(h.t, err)
}

func (h *syncHarness) boardPriority(rel, priority string) {
	h.t.Helper()
	_, err := h.tasks.UpdateTask(context.Background(), h.taskFor(rel).ID, updatePriority(priority))
	require.NoError(h.t, err)
}

// boardReorder is a person dragging cards so that the plan tasks of a step
// appear in the order of rels.
func (h *syncHarness) boardReorder(rels ...string) {
	h.t.Helper()
	h.tasks.mu.Lock()
	defer h.tasks.mu.Unlock()
	for i, rel := range rels {
		id := h.tasks.byExternalIDLocked(defaultExternalID(testRepoID, rel))
		require.NotEmpty(h.t, id, rel)
		h.tasks.tasks[id].Position = i
	}
}

// boardOrder returns the rel paths of the plan tasks of a step in board order.
func (h *syncHarness) boardOrder(stepID string) []string {
	h.t.Helper()
	tasks, err := h.tasks.ListTasks(context.Background(), "wf-1")
	require.NoError(h.t, err)
	var rels []string
	for _, t := range admittedBands(tasks)[stepID] {
		rels = append(rels, t.ExternalID[len("plan-file:"+testRepoID+":"):])
	}
	return rels
}
