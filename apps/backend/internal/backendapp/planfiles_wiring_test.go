package backendapp

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/kandev/kandev/internal/auth/authn"
	"github.com/kandev/kandev/internal/common/config"
	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/db"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

func planFilesTestDB(t *testing.T) *db.Pool {
	t.Helper()
	dbConn, err := db.OpenSQLite(filepath.Join(t.TempDir(), "planfiles.db"))
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbConn, "sqlite3")
	t.Cleanup(func() {
		if err := sqlxDB.Close(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	return db.NewPool(sqlxDB, sqlxDB)
}

// @covers AC-TASKS-PLAN-FILES-005.6
func TestInitPlanFilesService_FlagOffLeavesServiceNilButCreatesTables(t *testing.T) {
	harness := newBootStateTestHarness(t)
	pool := planFilesTestDB(t)
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "console"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	cfg := &config.Config{}

	svc, err := initPlanFilesService(cfg, pool, harness.taskSvc, harness.workflowSvc, log)

	if err != nil {
		t.Fatalf("initPlanFilesService: %v", err)
	}
	if svc != nil {
		t.Fatal("plan files service must stay nil while features.planFiles is off")
	}
	for _, table := range []string{"plan_file_configs", "plan_file_tasks"} {
		if _, err := pool.Writer().Exec("SELECT 1 FROM " + table); err != nil {
			t.Errorf("table %s must exist even with the flag off: %v", table, err)
		}
	}
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestInitPlanFilesService_WiresRealWorkspaceAuthorization(t *testing.T) {
	harness := newBootStateTestHarness(t)
	ctx := context.Background()
	ownerCtx := authn.WithIdentity(ctx, authn.Identity{UserID: "owner-1", Role: authn.RoleMember})
	workspace, err := harness.taskSvc.CreateWorkspace(ownerCtx, &taskservice.CreateWorkspaceRequest{Name: "plans workspace"})
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	log, err := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "console"})
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	cfg := &config.Config{}
	cfg.Features.PlanFiles = true

	svc, err := initPlanFilesService(cfg, planFilesTestDB(t), harness.taskSvc, harness.workflowSvc, log)

	if err != nil || svc == nil {
		t.Fatalf("initPlanFilesService = %v, %v; want a service", svc, err)
	}
	if _, err := svc.GetConfig(ownerCtx, workspace.ID); err != nil {
		t.Fatalf("owner GetConfig: %v", err)
	}
	foreignCtx := authn.WithIdentity(ctx, authn.Identity{UserID: "attacker-1", Role: authn.RoleMember})
	if _, err := svc.GetConfig(foreignCtx, workspace.ID); !errors.Is(err, repoerrors.ErrWorkspaceNotFound) {
		t.Fatalf("foreign identity GetConfig err = %v, want ErrWorkspaceNotFound", err)
	}
	if _, err := svc.GetConfig(ctx, workspace.ID); err != nil {
		t.Fatalf("identity-free GetConfig: %v", err)
	}
}
