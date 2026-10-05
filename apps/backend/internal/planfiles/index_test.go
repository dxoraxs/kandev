package planfiles

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	indexName = "INDEX.md"
	indexRel  = "docs/plans/INDEX.md"
	emDash    = "\u2014"
)

func indexCount(outcome string) int64 {
	if v, ok := indexTotal.Get("outcome=" + outcome).(interface{ Value() int64 }); ok {
		return v.Value()
	}
	return 0
}

func (h *syncHarness) setIndexFile(name string) {
	h.t.Helper()
	req := validRequest()
	req.IndexFile = &name
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(h.t, err)
}

func (h *syncHarness) indexErrors(summary PassSummary) []FileErrorRow {
	var out []FileErrorRow
	for _, fe := range summary.FileErrors {
		if fe.Reason == ReasonIndexNotOwned || fe.RelPath == indexRel {
			out = append(out, fe)
		}
	}
	return out
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.1
func TestIndex_ListsPlansGroupedByStatusInBoardOrder(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/b.md", planDoc("queued", "Second plan", "order: 20", "priority: high", "executor: codex"))
	h.writeFile("docs/plans/a.md", planDoc("queued", "First plan", "order: 10"))
	h.writeFile("docs/plans/c.md", planDoc("done", "Finished"))
	h.writeFile("docs/plans/hidden.md", planDoc("hidden", "Hidden one"))

	h.sync()

	want := indexMarker + "\n\n# Plans\n\n" +
		"## queued\n\n| Plan | Priority | Executor | Date |\n| --- | --- | --- | --- |\n" +
		"| [First plan](a.md) | medium |  |  |\n" +
		"| [Second plan](b.md) | high | codex |  |\n\n" +
		"## done\n\n| Plan | Priority | Executor | Date |\n| --- | --- | --- | --- |\n" +
		"| [Finished](c.md) | medium |  |  |\n"
	assert.Equal(t, want, h.read(indexRel))
	assert.NotContains(t, h.read(indexRel), emDash)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.1
func TestIndex_RowsFollowTheBoardOrderAfterAReorder(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("queued", "Alpha", "order: 10"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Beta", "order: 20"))
	h.sync()
	h.boardReorder("docs/plans/b.md", "docs/plans/a.md")

	h.sync()

	body := h.read(indexRel)
	assert.Less(t, strings.Index(body, "[Beta]"), strings.Index(body, "[Alpha]"), body)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.1
func TestIndex_OneIndexPerDirectoryWithPlans(t *testing.T) {
	h := newSyncHarness(t)
	req := validRequest()
	req.Directories = []string{"docs/plans", "docs/other", "docs/empty"}
	req.IndexFile = ptr(indexName)
	_, err := h.svc.PutConfig(context.Background(), testWorkspace, req)
	require.NoError(t, err)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile("docs/other/z.md", planDoc("queued", "Z"))
	h.writeFile("docs/empty/notes.md", "# not a plan\n")

	h.sync()

	assert.Contains(t, h.read("docs/plans/INDEX.md"), "[A](a.md)")
	assert.NotContains(t, h.read("docs/plans/INDEX.md"), "Z")
	assert.Contains(t, h.read("docs/other/INDEX.md"), "[Z](z.md)")
	_, statErr := os.Stat(filepath.Join(h.root, "docs/empty/INDEX.md"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestIndex_OffWritesNothing(t *testing.T) {
	h := newSyncHarness(t)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))

	h.sync()

	entries, err := os.ReadDir(filepath.Join(h.root, "docs/plans"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.3
func TestIndex_SecondPassOverUnchangedPlansDoesNotWrite(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	written := indexCount("written")
	h.sync()
	require.Equal(t, written+1, indexCount("written"))
	snap := h.snapshot(indexRel)

	h.sync()

	h.assertUnchanged(snap)
	assert.Equal(t, written+1, indexCount("written"), "no second write")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.3
func TestIndex_ChangedPlanRewritesTheIndex(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()

	h.writeFile("docs/plans/a.md", planDoc("done", "A"))
	h.sync()

	assert.Contains(t, h.read(indexRel), "## done")
	assert.NotContains(t, h.read(indexRel), "## queued")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestIndex_ForeignFileIsKeptAndReported(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.writeFile(indexRel, "# My own index\n")
	snap := h.snapshot(indexRel)
	before := indexCount("not_owned")

	summary := h.sync()

	h.assertUnchanged(snap)
	assert.Equal(t, OutcomePartial, summary.Outcome)
	require.Len(t, h.indexErrors(summary), 1)
	fe := h.indexErrors(summary)[0]
	assert.Equal(t, ReasonIndexNotOwned, fe.Reason)
	assert.Equal(t, indexRel, fe.RelPath)
	assert.Equal(t, testRepoID, fe.RepositoryID)
	assert.Equal(t, before+1, indexCount("not_owned"))
	assert.NotNil(t, h.taskFor("docs/plans/a.md"), "the plan itself still syncs")
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.2
func TestIndex_SymlinkAtTheIndexPathIsNotFollowed(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	outside := filepath.Join(t.TempDir(), "secret.md")
	require.NoError(t, os.WriteFile(outside, []byte(indexMarker+"\nsecret"), 0o644))
	require.NoError(t, os.Symlink(outside, filepath.Join(h.root, filepath.FromSlash(indexRel))))

	summary := h.sync()

	data, err := os.ReadFile(outside)
	require.NoError(t, err)
	assert.Equal(t, indexMarker+"\nsecret", string(data))
	assert.Equal(t, OutcomePartial, summary.Outcome)
	assert.NotEmpty(t, h.indexErrors(summary))
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.1
func TestIndex_EscapesMarkdownSpecialCharacters(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a b(1).md", planDoc("queued", "Odd", `title: "A | B [x] `+"`code`"+` \\ end\nnext line"`, "executor: \"ex|ec\""))

	h.sync()

	body := h.read(indexRel)
	var rows []string
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "| [") {
			rows = append(rows, line)
		}
	}
	require.Len(t, rows, 1, body)
	row := rows[0]
	assert.Contains(t, row, "A \\| B \\[x\\] \\`code\\` \\\\ end next line")
	assert.Contains(t, row, "(a%20b%281%29.md)")
	assert.Contains(t, row, `ex\|ec`)
	assert.Equal(t, 5, len(splitCells(row)), "five pipe separators, so four cells: %s", row)
}

// splitCells splits a table row on unescaped pipes.
func splitCells(row string) []string {
	var cells []string
	start := 0
	for i := 0; i < len(row); i++ {
		switch row[i] {
		case '\\':
			i++
		case '|':
			cells = append(cells, row[start:i])
			start = i + 1
		}
	}
	return cells
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.4
func TestIndex_IsNeitherAPlanFileNorUnadapted(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("queued", "A"))
	h.sync()
	summary := h.sync()

	assert.Equal(t, 0, summary.Counts.Unadapted)
	assert.Empty(t, summary.FileErrors)
	assert.Nil(t, h.tasks.byExternalID(defaultExternalID(testRepoID, indexRel)))
	rows, err := h.svc.UnadaptedCounts(context.Background(), testWorkspace, "")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.4
func TestIndex_NamedFileWithPlanFrontmatterIsNotAPlan(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile(indexRel, planDoc("queued", "Looks like a plan"))

	summary := h.sync()

	assert.Nil(t, h.tasks.byExternalID(defaultExternalID(testRepoID, indexRel)))
	assert.Equal(t, 0, summary.Counts.Created)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.4
func TestIndex_UnadaptedCountsSkipTheIndexFileOnly(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile(indexRel, "# hand written\n")
	h.writeFile("docs/plans/notes.md", "# notes\n")

	rows, err := h.svc.UnadaptedCounts(context.Background(), testWorkspace, "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].Count)
}

// @covers AC-TASKS-PLAN-BOARD-OPS-006.3
func TestIndex_UnorderedPlansListInPathOrderWhateverTheCreationOrder(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/c.md", planDoc("queued", "Cee"))
	h.writeFile("docs/plans/a.md", planDoc("queued", "Aye"))
	h.writeFile("docs/plans/b.md", planDoc("queued", "Bee"))

	h.sync()
	first := h.read(indexRel)

	body := first
	assert.Less(t, strings.Index(body, "[Aye]"), strings.Index(body, "[Bee]"))
	assert.Less(t, strings.Index(body, "[Bee]"), strings.Index(body, "[Cee]"))
	assert.NotContains(t, body, emDash)
}

// @covers AC-TASKS-PLAN-CARD-001.1
func TestIndex_DateColumnShowsTheValidDateAsWritten(t *testing.T) {
	h := newSyncHarness(t)
	h.setIndexFile(indexName)
	h.writeFile("docs/plans/a.md", planDoc("waiting_owner", "Dated", "date: 2026-10-12"))
	h.writeFile("docs/plans/b.md", planDoc("waiting_owner", "Broken", "date: 2026-02-30"))
	h.writeFile("docs/plans/c.md", planDoc("waiting_owner", "Undated"))

	h.sync()

	got := h.read(indexRel)
	assert.Contains(t, got, "| [Dated](a.md) | medium |  | 2026-10-12 |\n")
	assert.Contains(t, got, "| [Broken](b.md) | medium |  |  |\n")
	assert.Contains(t, got, "| [Undated](c.md) | medium |  |  |\n")
}
