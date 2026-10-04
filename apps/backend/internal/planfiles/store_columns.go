package planfiles

import (
	"fmt"

	"github.com/kandev/kandev/internal/db"
)

// addedColumn is a column that was added to a table after its first release.
type addedColumn struct {
	table string
	name  string
	ddl   string
}

// addedColumns lists every column that CREATE TABLE IF NOT EXISTS cannot add
// to a database created before it existed.
var addedColumns = []addedColumn{
	{"plan_file_configs", "executor_steps", "TEXT NOT NULL DEFAULT '{}'"},
	{"plan_file_configs", "notes_heading", "TEXT NOT NULL DEFAULT ''"},
	{"plan_file_configs", "wake_on_date", "INTEGER NOT NULL DEFAULT 1"},
	{"plan_file_configs", "stale_after_days", "INTEGER NOT NULL DEFAULT 7"},
	{"plan_file_configs", "index_file", "TEXT NOT NULL DEFAULT ''"},
	{"plan_file_tasks", "synced_depends_on", "TEXT NOT NULL DEFAULT '[]'"},
}

// addMissingColumns adds each addedColumns entry the table does not declare.
// A concurrent or replayed add is treated as done.
func (s *Store) addMissingColumns() error {
	existing := map[string]map[string]bool{}
	for _, col := range addedColumns {
		cols, ok := existing[col.table]
		if !ok {
			var err error
			if cols, err = db.TableColumns(s.db, col.table); err != nil {
				return fmt.Errorf("read %s columns: %w", col.table, err)
			}
			existing[col.table] = cols
		}
		if cols[col.name] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", col.table, col.name, col.ddl)
		if _, err := s.db.Exec(stmt); err != nil && !db.IsDuplicateColumnError(err) {
			return fmt.Errorf("add %s.%s: %w", col.table, col.name, err)
		}
	}
	return nil
}
