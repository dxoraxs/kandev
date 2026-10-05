package format

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	keyDate = "date"
	// DateLayout is the layout of the `date` value: a real calendar date.
	DateLayout = "2006-01-02"
)

// decodeDate reads the optional `date` key. A value that is not a calendar
// date in YYYY-MM-DD form is reported as invalid_date and leaves Date empty.
func (pf *PlanFile) decodeDate(fields map[string]yaml.Node) {
	n, ok := fields[keyDate]
	if !ok || isNull(n) {
		return
	}
	s, err := scalarString(n)
	if err != nil {
		pf.fail(keyDate, fmt.Errorf("invalid_date: %v", err))
		return
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	if _, err := time.Parse(DateLayout, s); err != nil {
		pf.fail(keyDate, fmt.Errorf("invalid_date: must be a calendar date in YYYY-MM-DD form"))
		return
	}
	pf.Date = s
}
