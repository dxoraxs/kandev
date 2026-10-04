package planfiles

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/kandev/kandev/internal/planfiles/format"
	"github.com/kandev/kandev/internal/task/contract"
)

func filePlan(board format.BoardStatus, title, executor string) format.PlanFile {
	return format.PlanFile{Board: board, Title: title, Executor: executor, Priority: "medium"}
}

// @covers AC-TASKS-PLAN-FILES-002.3
func TestProjectTitle_ExecutorPrefixExceptDone(t *testing.T) {
	assert.Equal(t, "[codex] Build it", projectTitle(filePlan(format.BoardQueued, "Build it", "codex")))
	assert.Equal(t, "Build it", projectTitle(filePlan(format.BoardDone, "Build it", "codex")))
	assert.Equal(t, "Build it", projectTitle(filePlan(format.BoardQueued, "Build it", "")))
	// An unreadable board value is not done, so the prefix stays.
	assert.Equal(t, "[codex] Build it", projectTitle(filePlan("", "Build it", "codex")))
}

// @covers AC-TASKS-PLAN-FILES-002.3
func TestProjectTitle_TruncatesToLimitWithEllipsis(t *testing.T) {
	long := strings.Repeat("я", 90)

	got := projectTitle(filePlan(format.BoardQueued, long, "codex"))

	assert.Equal(t, contract.TaskTitleMaxLength, utf8.RuneCountInString(got))
	assert.True(t, strings.HasSuffix(got, "…"))
	assert.True(t, strings.HasPrefix(got, "[codex] я"))
}

func TestProjectTitle_CollapsesLineBreaks(t *testing.T) {
	assert.Equal(t, "One Two", projectTitle(filePlan(format.BoardQueued, "One\n  Two", "")))
}

// @covers AC-TASKS-PLAN-FILES-002.4
func TestProjectDescription_HeaderThenBody(t *testing.T) {
	got := projectDescription(descriptionInput{
		RepositoryName: "city_companion",
		RelPath:        "docs/plans/a.md",
		File:           format.PlanFile{Executor: "codex", Body: "\n# Heading\n\nText\n"},
		Dependencies:   []dependencyLink{{Name: "b.md", TaskID: "task-b"}},
		Notice:         "board edit not saved",
	})

	want := "> Plan file: `city_companion` `docs/plans/a.md`\n" +
		"> Executor: codex\n" +
		"> Depends on: [b.md](/t/task-b)\n" +
		"> Notice: board edit not saved\n" +
		"\n# Heading\n\nText\n"
	assert.Equal(t, want, got)
}

// @covers AC-TASKS-PLAN-FILES-001.4
func TestProjectDescription_ShowsParseErrorsOnOneLineEach(t *testing.T) {
	got := projectDescription(descriptionInput{
		RepositoryName: "r",
		RelPath:        "docs/plans/a.md",
		File: format.PlanFile{
			ParseErrors: []string{"board: unknown status \"x\"\ninjected line", "order: must be a number"},
			Body:        "body",
		},
	})

	assert.Contains(t, got, "> Parse error: board: unknown status \"x\" injected line\n")
	assert.Contains(t, got, "> Parse error: order: must be a number\n")
	assert.True(t, strings.HasSuffix(got, "\nbody"))
}

func TestOrderKey_SortsByOrderThenPathUnorderedLast(t *testing.T) {
	ten, two := 10.0, 2.0
	keys := []orderKey{
		newOrderKey(nil, "a.md", "r1"),
		newOrderKey(&ten, "z.md", "r1"),
		newOrderKey(&two, "m.md", "r1"),
		newOrderKey(nil, "b.md", "r1"),
	}

	assert.True(t, keys[2].less(keys[1]), "2 before 10")
	assert.True(t, keys[1].less(keys[0]), "ordered before unordered")
	assert.True(t, keys[0].less(keys[3]), "unordered fall back to path")
	assert.False(t, keys[3].less(keys[0]))
}

func TestOrderKey_StringIsStable(t *testing.T) {
	ten := 10.5

	assert.Equal(t, "0:10.5:docs/plans/a.md", newOrderKey(&ten, "docs/plans/a.md", "r1").String())
	assert.Equal(t, "1::docs/plans/b.md", newOrderKey(nil, "docs/plans/b.md", "r1").String())
}

func TestDefaultExternalID_UsesRepositoryAndSlashedPath(t *testing.T) {
	assert.Equal(t, "plan-file:repo-1:docs/plans/a.md", defaultExternalID("repo-1", "docs/plans/a.md"))
}

func TestDependencyPath_ResolvesInSameDirectory(t *testing.T) {
	assert.Equal(t, "docs/plans/b.md", dependencyPath("docs/plans/a.md", "b.md"))
	assert.Equal(t, "docs/plans/b.md", dependencyPath("docs/plans/a.md", "./b.md"))
	assert.Equal(t, "", dependencyPath("docs/plans/a.md", "../x.md"))
	assert.Equal(t, "", dependencyPath("docs/plans/a.md", "sub/b.md"))
}
