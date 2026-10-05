package format

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kandev/kandev/internal/task/models"
)

// NewPlanInput is the content of a new plan file. Priority is expected to be
// validated by the caller; an empty one means the default.
type NewPlanInput struct {
	Title    string
	Priority string
	Executor string
	Body     string
}

// NewPlan renders a new plan file: frontmatter with `board: queued`, the
// title, the priority, and the executor when given, then a level-one heading
// with the title and the body. Title and executor are YAML double-quoted
// scalars, so Parse returns the same values.
func NewPlan(in NewPlanInput) []byte {
	title := strings.TrimSpace(in.Title)
	priority := strings.TrimSpace(in.Priority)
	if priority == "" {
		priority = models.TaskPriorityMedium
	}
	var b strings.Builder
	b.WriteString("---\nboard: queued\n")
	b.WriteString(keyTitle + ": " + quoteYAML(title) + "\n")
	b.WriteString(keyPriority + ": " + priority + "\n")
	if executor := strings.TrimSpace(in.Executor); executor != "" {
		b.WriteString(keyExecutor + ": " + quoteYAML(executor) + "\n")
	}
	b.WriteString("---\n# " + singleLine(title) + "\n")
	if body := in.Body; strings.TrimSpace(body) != "" {
		b.WriteString("\n" + body)
		if !strings.HasSuffix(body, "\n") {
			b.WriteString("\n")
		}
	}
	return []byte(b.String())
}

// quoteYAML renders s as a YAML double-quoted scalar. Go's escapes for
// printable and control characters are a subset of the YAML ones.
func quoteYAML(s string) string {
	return strconv.Quote(strings.ToValidUTF8(s, "�"))
}

// WorkOrderDone reports whether the frontmatter `status` scalar is `done`.
func WorkOrderDone(content []byte) bool {
	fm, ok := locate(content)
	if !ok {
		return false
	}
	var fields map[string]yaml.Node
	if err := yaml.Unmarshal(fm.yamlText(), &fields); err != nil {
		return false
	}
	n, ok := fields["status"]
	if !ok || n.Kind != yaml.ScalarNode {
		return false
	}
	return n.Value == "done"
}

func singleLine(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}
