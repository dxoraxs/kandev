package planfiles

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// @covers AC-TASKS-PLAN-FILES-002.4
func TestProjectDescription_CapsLongBodyAtLineBoundary(t *testing.T) {
	line := strings.Repeat("я", 99) + "\n"
	body := strings.Repeat(line, 2*maxDescriptionBodyBytes/len(line))

	got := projectDescription(descriptionInput{
		RepositoryName: "r",
		RelPath:        "docs/plans/a.md",
		File:           format.PlanFile{Body: body},
	})

	header, kept, ok := strings.Cut(got, "\n\n")
	assert.True(t, ok)
	assert.Contains(t, header, "> Truncated:")
	assert.LessOrEqual(t, len(kept), maxDescriptionBodyBytes)
	assert.True(t, utf8.ValidString(kept))
	assert.True(t, strings.HasSuffix(kept, "\n"), "cut must land on a line break")
	assert.True(t, strings.HasPrefix(body, kept))
}

// @covers AC-TASKS-PLAN-FILES-002.4
func TestProjectDescription_CapsBodyWithoutLineBreaksOnRuneBoundary(t *testing.T) {
	body := strings.Repeat("я", maxDescriptionBodyBytes)

	got := projectDescription(descriptionInput{RepositoryName: "r", RelPath: "a.md", File: format.PlanFile{Body: body}})

	_, kept, _ := strings.Cut(got, "\n\n")
	assert.LessOrEqual(t, len(kept), maxDescriptionBodyBytes)
	assert.Greater(t, len(kept), maxDescriptionBodyBytes-utf8.UTFMax)
	assert.True(t, utf8.ValidString(kept))
}

func TestProjectDescription_ShortBodyHasNoTruncationNote(t *testing.T) {
	got := projectDescription(descriptionInput{RepositoryName: "r", RelPath: "a.md", File: format.PlanFile{Body: "short"}})

	assert.NotContains(t, got, "Truncated")
}
