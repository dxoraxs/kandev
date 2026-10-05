package planfiles

import (
	"expvar"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
)

// expvar maps published at package init, exposed through /debug/vars in dev
// mode. Labels come from closed sets (pass outcome, failure reason); no
// workspace, task, repository, or path value is ever a label.
var (
	passTotal       = expvar.NewMap("plan_files_pass_total")
	fileErrorsTotal = expvar.NewMap("plan_files_file_errors_total")
	writebackTotal  = expvar.NewMap("plan_files_writeback_total")
	decisionTotal   = expvar.NewMap("plan_files_decision_total")
	commitTotal     = expvar.NewMap("plan_files_commit_total")
	indexTotal      = expvar.NewMap("plan_files_index_total")
	wakeTotal       = expvar.NewInt("plan_files_wake_total")
)

// Write-back outcomes. The set is closed; it is also the label set of
// plan_files_writeback_total.
const (
	writebackWritten        = "written"
	writebackConflict       = "conflict"
	writebackFailed         = "failed"
	writebackHandoffSkipped = "handoff_skipped"
)

func incPass(log *logger.Logger, outcome string) {
	passTotal.Add("outcome="+outcome, 1)
	log.Info("plan_files.metric.pass", zap.String("outcome", outcome))
}

func incFileError(log *logger.Logger, reason string) {
	fileErrorsTotal.Add("reason="+reason, 1)
	log.Info("plan_files.metric.file_error", zap.String("reason", reason))
}

func incWriteback(log *logger.Logger, outcome string) {
	writebackTotal.Add("outcome="+outcome, 1)
	log.Info("plan_files.metric.writeback", zap.String("outcome", outcome))
}

// incDecision counts one applied owner decision. action is one of the closed
// set accept, return.
func incDecision(log *logger.Logger, action string) {
	decisionTotal.Add("action="+action, 1)
	log.Info("plan_files.metric.decision", zap.String("action", action))
}

// Commit outcomes. The set is closed; it is also the label set of
// plan_files_commit_total.
const (
	commitCommitted = "committed"
	commitBusy      = "repository_busy"
	commitNothing   = "nothing_to_commit"
	commitFailed    = "commit_failed"
)

func incCommit(log *logger.Logger, outcome string) {
	commitTotal.Add("outcome="+outcome, 1)
	log.Info("plan_files.metric.commit", zap.String("outcome", outcome))
}

// Index outcomes. The set is closed; it is also the label set of
// plan_files_index_total.
const (
	indexWritten  = "written"
	indexNotOwned = "not_owned"
)

func incIndex(log *logger.Logger, outcome string) {
	indexTotal.Add("outcome="+outcome, 1)
	log.Info("plan_files.metric.index", zap.String("outcome", outcome))
}

// incWake counts one written date wake-up.
func incWake(log *logger.Logger) {
	wakeTotal.Add(1)
	log.Info("plan_files.metric.wake")
}
