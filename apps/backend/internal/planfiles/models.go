// Package planfiles reconciles plan files in a workspace's local repositories
// with tasks on a plan board. This file holds the persisted and wire shapes.
package planfiles

import (
	"time"

	"github.com/kandev/kandev/internal/planfiles/format"
)

// MaxStoredFileErrors caps the per-file error rows kept on the config row.
const MaxStoredFileErrors = 200

// DefaultDirectories returns the repository-relative directories scanned when
// the owner has not chosen others.
func DefaultDirectories() []string {
	return []string{"docs/plans", "docs/superpowers/plans"}
}

// PassCounts summarises one sync pass.
type PassCounts struct {
	Created    int `json:"created"`
	Updated    int `json:"updated"`
	Moved      int `json:"moved"`
	Archived   int `json:"archived"`
	Unarchived int `json:"unarchived"`
	Failed     int `json:"failed"`
	// Unadapted counts Markdown files in scanned directories that carry no
	// board key. They are neither errors nor tasks.
	Unadapted int `json:"unadapted"`
}

// UnadaptedRepo is one local repository holding Markdown files without a board
// key. Directories are the scanned directories that contain such files.
type UnadaptedRepo struct {
	RepositoryID   string   `json:"repository_id"`
	RepositoryName string   `json:"repository_name"`
	Count          int      `json:"count"`
	Directories    []string `json:"directories"`
}

// FileErrorRow is one file (or repository) that a pass could not process.
type FileErrorRow struct {
	RepositoryID   string `json:"repository_id"`
	RepositoryName string `json:"repository_name"`
	RelPath        string `json:"rel_path"`
	Reason         string `json:"reason"`
}

// Config is the per-workspace plan-file configuration plus the status of the
// last sync pass.
type Config struct {
	WorkspaceID string `json:"workspace_id"`
	Enabled     bool   `json:"enabled"`
	WorkflowID  string `json:"workflow_id"`
	// StatusSteps maps every board status except hidden to a step ID of
	// WorkflowID.
	StatusSteps map[format.BoardStatus]string `json:"status_steps"`
	// Directories are repository-relative directories scanned for plan files.
	Directories []string `json:"directories"`
	// ExecutorSteps maps a step ID of WorkflowID to an executor name. A step
	// listed here is never a status step.
	ExecutorSteps map[string]string `json:"executor_steps"`
	// NotesHeading is the owner-notes heading; empty reads as the default.
	NotesHeading string `json:"notes_heading"`
	// WakeOnDate moves a waiting plan to the owner when its date arrives.
	WakeOnDate bool `json:"wake_on_date"`
	// StaleAfterDays flags an in-progress plan untouched this long; 0 is off.
	StaleAfterDays int `json:"stale_after_days"`
	// IndexFile is the generated index file name; empty is off.
	IndexFile      string         `json:"index_file"`
	LastPassAt     *time.Time     `json:"last_pass_at,omitempty"`
	LastPassOK     bool           `json:"last_pass_ok"`
	LastCounts     PassCounts     `json:"last_counts"`
	LastFileErrors []FileErrorRow `json:"last_file_errors"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// TaskRow records the state the last pass applied to one plan task. A
// divergence between the live task and the synced_* values marks a board edit.
type TaskRow struct {
	TaskID         string `json:"task_id"`
	WorkspaceID    string `json:"workspace_id"`
	RepositoryID   string `json:"repository_id"`
	RelPath        string `json:"rel_path"`
	ExternalID     string `json:"external_id"`
	ContentHash    string `json:"content_hash"`
	SyncedStepID   string `json:"synced_step_id"`
	SyncedPriority string `json:"synced_priority"`
	SyncedOrderKey string `json:"synced_order_key"`
	// SyncedDependsOn is the sorted blocker task IDs the last pass applied.
	SyncedDependsOn []string  `json:"synced_depends_on"`
	Notice          string    `json:"notice"`
	LastSeenAt      time.Time `json:"last_seen_at"`
}

// PutConfigRequest is the body of PUT /api/v1/plan-files/config. The pointer
// fields are optional: an absent field keeps the stored value, or the default
// when no config is stored yet.
type PutConfigRequest struct {
	Enabled        bool                          `json:"enabled"`
	WorkflowID     string                        `json:"workflow_id"`
	StatusSteps    map[format.BoardStatus]string `json:"status_steps"`
	Directories    []string                      `json:"directories"`
	ExecutorSteps  *map[string]string            `json:"executor_steps,omitempty"`
	NotesHeading   *string                       `json:"notes_heading,omitempty"`
	WakeOnDate     *bool                         `json:"wake_on_date,omitempty"`
	StaleAfterDays *int                          `json:"stale_after_days,omitempty"`
	IndexFile      *string                       `json:"index_file,omitempty"`
}

// CreateBoardResult is the response of POST /api/v1/plan-files/board.
type CreateBoardResult struct {
	WorkflowID  string                        `json:"workflow_id"`
	StatusSteps map[format.BoardStatus]string `json:"status_steps"`
}
