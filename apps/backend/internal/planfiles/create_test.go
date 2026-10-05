package planfiles

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

func createHarness(t *testing.T) *syncHarness {
	t.Helper()
	h := newSyncHarness(t)
	h.svc.SetClock(func() time.Time { return time.Date(2026, 10, 5, 9, 7, 0, 0, time.Local) })
	return h
}

func (h *syncHarness) create(req CreatePlanRequest) (CreatePlanResult, error) {
	h.t.Helper()
	if req.RepositoryID == "" {
		req.RepositoryID = testRepoID
	}
	if req.Directory == "" {
		req.Directory = plansDir
	}
	return h.svc.CreatePlan(context.Background(), testWorkspace, req)
}

func requireCreateCode(t *testing.T, err error, code string) {
	t.Helper()
	var createErr *CreatePlanError
	require.True(t, errors.As(err, &createErr), "got %v", err)
	assert.Equal(t, code, createErr.Code)
}

func (h *syncHarness) planDirEntries() []string {
	h.t.Helper()
	entries, err := os.ReadDir(filepath.Join(h.root, filepath.FromSlash(plansDir)))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(h.t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.2
// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
// @covers AC-TASKS-PLAN-BOARD-OPS-007.4
func TestCreatePlan_NonAsciiTitleGetsATimestampNameAndTheTaskIsQueued(t *testing.T) {
	h := createHarness(t)

	result, err := h.create(CreatePlanRequest{Title: "Заказ в чате"})

	require.NoError(t, err)
	assert.Equal(t, "docs/plans/plan-20261005-0907.md", result.RelPath)
	assert.Equal(t, testRepoID, result.RepositoryID)
	task := h.taskFor(result.RelPath)
	assert.Equal(t, task.ID, result.TaskID)
	assert.Equal(t, h.step(format.BoardQueued), task.WorkflowStepID)
	pf, ok := format.Parse("plan-20261005-0907.md", []byte(h.read(result.RelPath)))
	require.True(t, ok)
	assert.Empty(t, pf.ParseErrors)
	assert.Equal(t, "Заказ в чате", pf.Title)
	assert.Equal(t, format.BoardQueued, pf.Board)
	assert.Equal(t, taskmodels.TaskPriorityMedium, pf.Priority)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.2
func TestCreatePlan_SlugOfTheTitleNamesTheFile(t *testing.T) {
	h := createHarness(t)

	result, err := h.create(CreatePlanRequest{Title: "Fix: totals #2", Priority: "high", Executor: "codex", Body: "Details.\n"})

	require.NoError(t, err)
	assert.Equal(t, "docs/plans/fix-totals-2.md", result.RelPath)
	assert.Equal(t, "---\nboard: queued\ntitle: \"Fix: totals #2\"\npriority: high\nexecutor: \"codex\"\n---\n# Fix: totals #2\n\nDetails.\n",
		h.read(result.RelPath))
	pf, ok := format.Parse("fix-totals-2.md", []byte(h.read(result.RelPath)))
	require.True(t, ok)
	assert.Equal(t, "Fix: totals #2", pf.Title)
	assert.Equal(t, "codex", pf.Executor)
	assert.Equal(t, "high", pf.Priority)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.2
func TestCreatePlan_SlugIsCutAt80BytesWithoutATrailingDash(t *testing.T) {
	h := createHarness(t)
	title := strings.Repeat("abcdefghi ", 12)

	result, err := h.create(CreatePlanRequest{Title: title})

	require.NoError(t, err)
	name := strings.TrimSuffix(strings.TrimPrefix(result.RelPath, "docs/plans/"), ".md")
	assert.LessOrEqual(t, len(name), 80)
	assert.False(t, strings.HasSuffix(name, "-"), name)
	assert.True(t, strings.HasPrefix(name, "abcdefghi-abcdefghi-"), name)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.2
func TestCreatePlan_GivenFileNameIsUsedAsIs(t *testing.T) {
	h := createHarness(t)

	result, err := h.create(CreatePlanRequest{Title: "Anything", FileName: " My_Plan.v2.md "})

	require.NoError(t, err)
	assert.Equal(t, "docs/plans/My_Plan.v2.md", result.RelPath)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreatePlan_ExistingFileIsKeptAndReported(t *testing.T) {
	h := createHarness(t)
	first, err := h.create(CreatePlanRequest{Title: "Same", Body: "first"})
	require.NoError(t, err)
	before := h.read(first.RelPath)

	_, err = h.create(CreatePlanRequest{Title: "Same", Body: "second"})

	requireCreateCode(t, err, CodeFileExists)
	assert.Equal(t, before, h.read(first.RelPath))
	assert.Equal(t, []string{"same.md"}, h.planDirEntries())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreatePlan_ExistingFileWithoutBoardKeyIsNotReplaced(t *testing.T) {
	h := createHarness(t)
	h.writeFile("docs/plans/notes.md", "# Notes\n")

	_, err := h.create(CreatePlanRequest{Title: "Notes"})

	requireCreateCode(t, err, CodeFileExists)
	assert.Equal(t, "# Notes\n", h.read("docs/plans/notes.md"))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.2
func TestCreatePlan_InvalidInputIsRefusedAndNothingIsWritten(t *testing.T) {
	long := strings.Repeat("a", 118) + ".md"
	cases := map[string]CreatePlanRequest{
		"blank title":           {Title: "   "},
		"title too long":        {Title: strings.Repeat("t", 201)},
		"file name no ext":      {Title: "T", FileName: "plan"},
		"file name leading dot": {Title: "T", FileName: ".hidden.md"},
		"file name dash first":  {Title: "T", FileName: "-x.md"},
		"file name with slash":  {Title: "T", FileName: "sub/x.md"},
		"file name dotdot":      {Title: "T", FileName: "../x.md"},
		"file name space":       {Title: "T", FileName: "my plan.md"},
		"file name non ascii":   {Title: "T", FileName: "план.md"},
		"file name too long":    {Title: "T", FileName: long + "x"},
		"file name uppercase":   {Title: "T", FileName: "x.MD"},
		"priority":              {Title: "T", Priority: "urgent"},
		"executor too long":     {Title: "T", Executor: strings.Repeat("e", format.ExecutorMaxBytes+1)},
		"executor line break":   {Title: "T", Executor: "a\nb"},
		"directory not scanned": {Title: "T", Directory: "docs/other"},
		"directory parent":      {Title: "T", Directory: "docs/plans/../../.."},
		"directory absolute":    {Title: "T", Directory: "/etc"},
		"body too large":        {Title: "T", Body: strings.Repeat("x", 1<<20)},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			h := createHarness(t)
			_, err := h.create(req)
			requireCreateCode(t, err, CodeInvalidPlan)
			assert.Empty(t, h.planDirEntries())
			_, statErr := os.Stat(filepath.Join(h.root, "etc"))
			assert.True(t, os.IsNotExist(statErr))
		})
	}
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.2
func TestCreatePlan_MaxLengthFileNameIsAccepted(t *testing.T) {
	h := createHarness(t)
	name := strings.Repeat("a", 117) + ".md"

	result, err := h.create(CreatePlanRequest{Title: "T", FileName: name})

	require.NoError(t, err)
	assert.Equal(t, "docs/plans/"+name, result.RelPath)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreatePlan_FileNameOfTheIndexFileIsRefused(t *testing.T) {
	h := createHarness(t)
	indexFile := "INDEX.md"
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, func() *PutConfigRequest {
		req := validRequest()
		req.IndexFile = &indexFile
		return req
	}())
	require.NoError(t, err)

	_, err = h.create(CreatePlanRequest{Title: "T", FileName: indexFile})

	requireCreateCode(t, err, CodeInvalidPlan)
	assert.Empty(t, h.planDirEntries())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.1
func TestCreatePlan_RepositoryMustBeALocalRepositoryOfTheWorkspace(t *testing.T) {
	h := createHarness(t)
	h.tasks.repos = append(h.tasks.repos,
		&taskmodels.Repository{ID: "remote-1", WorkspaceID: testWorkspace, Name: "remote", SourceType: "github"},
		&taskmodels.Repository{ID: "nopath", WorkspaceID: testWorkspace, Name: "nopath", SourceType: "local"})

	for _, id := range []string{"missing", "remote-1", "nopath"} {
		_, err := h.create(CreatePlanRequest{RepositoryID: id, Title: "T"})
		requireCreateCode(t, err, CodeRepositoryNotFound)
	}
	assert.Empty(t, h.planDirEntries())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreatePlan_SymlinkedDirectoryOutsideTheRepositoryIsRefused(t *testing.T) {
	h := createHarness(t)
	outside := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(h.root, "docs"), 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(h.root, "docs", "plans")))

	_, err := h.create(CreatePlanRequest{Title: "Escape"})

	requireCreateCode(t, err, CodeInvalidPlan)
	entries, readErr := os.ReadDir(outside)
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreatePlan_CreatesTheConfiguredDirectoryWhenMissing(t *testing.T) {
	h := createHarness(t)
	_, err := os.Stat(filepath.Join(h.root, "docs"))
	require.True(t, os.IsNotExist(err))

	result, err := h.create(CreatePlanRequest{Title: "First plan"})

	require.NoError(t, err)
	assert.Equal(t, "docs/plans/first-plan.md", result.RelPath)
	assert.Equal(t, h.step(format.BoardQueued), h.taskFor(result.RelPath).WorkflowStepID)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.4
func TestCreatePlan_WithoutAnEnabledConfigNothingIsWritten(t *testing.T) {
	h := createHarness(t)
	disabled := validRequest()
	disabled.Enabled = false
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, disabled)
	require.NoError(t, err)

	_, err = h.create(CreatePlanRequest{Title: "T"})

	assert.ErrorIs(t, err, ErrNotConfigured)
	assert.Empty(t, h.planDirEntries())
}

// @covers AC-TASKS-PLAN-BOARD-OPS-007.3
func TestCreatePlan_ConcurrentCreatesOfOneNameYieldExactlyOneFile(t *testing.T) {
	h := createHarness(t)
	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			_, err := h.svc.CreatePlan(context.Background(), testWorkspace, CreatePlanRequest{
				RepositoryID: testRepoID, Directory: plansDir, Title: "Race", FileName: "race.md", Body: strings.Repeat("x", i+1),
			})
			errs <- err
		}(i)
	}
	created := 0
	for i := 0; i < n; i++ {
		err := <-errs
		if err == nil {
			created++
			continue
		}
		var createErr *CreatePlanError
		require.True(t, errors.As(err, &createErr), "got %v", err)
		assert.Equal(t, CodeFileExists, createErr.Code)
	}
	assert.Equal(t, 1, created)
	assert.Equal(t, []string{"race.md"}, h.planDirEntries())
}
