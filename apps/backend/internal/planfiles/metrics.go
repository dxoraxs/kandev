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
