package planfiles

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/planfiles/format"
	taskmodels "github.com/kandev/kandev/internal/task/models"
	"github.com/kandev/kandev/internal/task/repository/repoerrors"
	taskservice "github.com/kandev/kandev/internal/task/service"
	wfmodels "github.com/kandev/kandev/internal/workflow/models"
)

const (
	testWorkspace = "ws-1"
	otherWS       = "ws-2"
)

// fakeWorkflows implements WorkflowAccess and StepAccess in memory.
type fakeWorkflows struct {
	t         *testing.T
	workflows map[string]*taskmodels.Workflow
	steps     map[string][]*wfmodels.WorkflowStep
	created   []*taskservice.CreateWorkflowRequest
	deleted   []string
	// dropSteps simulates a template whose steps failed to be created.
	dropSteps bool
}

func newFakeWorkflows(t *testing.T) *fakeWorkflows {
	return &fakeWorkflows{
		t:         t,
		workflows: map[string]*taskmodels.Workflow{},
		steps:     map[string][]*wfmodels.WorkflowStep{},
	}
}

func (f *fakeWorkflows) addWorkflow(id, workspaceID string, stepIDs ...string) {
	f.workflows[id] = &taskmodels.Workflow{ID: id, WorkspaceID: workspaceID}
	for i, stepID := range stepIDs {
		f.steps[id] = append(f.steps[id], &wfmodels.WorkflowStep{ID: stepID, WorkflowID: id, Position: i})
	}
}

func (f *fakeWorkflows) GetWorkflow(_ context.Context, id string) (*taskmodels.Workflow, error) {
	wf, ok := f.workflows[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", repoerrors.ErrWorkflowNotFound, id)
	}
	return wf, nil
}

func (f *fakeWorkflows) CreateWorkflow(_ context.Context, req *taskservice.CreateWorkflowRequest) (*taskmodels.Workflow, error) {
	f.created = append(f.created, req)
	id := fmt.Sprintf("created-%d", len(f.created))
	f.workflows[id] = &taskmodels.Workflow{ID: id, WorkspaceID: req.WorkspaceID, WorkflowTemplateID: req.WorkflowTemplateID}
	if !f.dropSteps {
		tmpl := loadPlansTemplate(f.t)
		for _, def := range tmpl.Steps {
			f.steps[id] = append(f.steps[id], &wfmodels.WorkflowStep{
				ID: id + "-" + def.ID, WorkflowID: id, Name: def.Name, Position: def.Position,
			})
		}
	}
	return f.workflows[id], nil
}

func (f *fakeWorkflows) DeleteWorkflow(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	delete(f.workflows, id)
	return nil
}

func (f *fakeWorkflows) ListStepsByWorkflow(_ context.Context, workflowID string) ([]*wfmodels.WorkflowStep, error) {
	return f.steps[workflowID], nil
}

func (f *fakeWorkflows) GetTemplate(_ context.Context, id string) (*wfmodels.WorkflowTemplate, error) {
	if id != PlansTemplateID {
		return nil, errors.New("unknown template")
	}
	return loadPlansTemplate(f.t), nil
}

func newTestService(t *testing.T) (*Service, *fakeWorkflows) {
	t.Helper()
	fake := newFakeWorkflows(t)
	fake.addWorkflow("wf-1", testWorkspace, "q", "ip", "wo", "we", "df", "dn", "extra")
	fake.addWorkflow("wf-foreign", otherWS, "fq", "fip", "fwo", "fwe", "fdf", "fdn")
	svc := NewService(setupTestStore(t), fake, fake, logger.Default())
	return svc, fake
}

func validRequest() *PutConfigRequest {
	return &PutConfigRequest{
		Enabled:    true,
		WorkflowID: "wf-1",
		StatusSteps: map[format.BoardStatus]string{
			format.BoardQueued:          "q",
			format.BoardInProgress:      "ip",
			format.BoardWaitingOwner:    "wo",
			format.BoardWaitingExternal: "we",
			format.BoardDeferred:        "df",
			format.BoardDone:            "dn",
		},
		Directories: []string{"docs/plans"},
	}
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestPutConfig_SavesValidConfig(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	saved, err := svc.PutConfig(ctx, testWorkspace, validRequest())
	require.NoError(t, err)
	assert.True(t, saved.Enabled)
	assert.Equal(t, "wf-1", saved.WorkflowID)
	assert.Equal(t, "dn", saved.StatusSteps[format.BoardDone])

	got, err := svc.GetConfig(ctx, testWorkspace)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, saved.Directories, got.Directories)
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestPutConfig_RejectsInvalidRequests(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(r *PutConfigRequest)
	}{
		{"foreign workflow", func(r *PutConfigRequest) {
			r.WorkflowID = "wf-foreign"
			r.StatusSteps = map[format.BoardStatus]string{
				format.BoardQueued: "fq", format.BoardInProgress: "fip", format.BoardWaitingOwner: "fwo",
				format.BoardWaitingExternal: "fwe", format.BoardDeferred: "fdf", format.BoardDone: "fdn",
			}
		}},
		{"unknown workflow", func(r *PutConfigRequest) { r.WorkflowID = "wf-missing" }},
		{"empty workflow", func(r *PutConfigRequest) { r.WorkflowID = "" }},
		{"step from another workflow", func(r *PutConfigRequest) { r.StatusSteps[format.BoardDone] = "fdn" }},
		{"missing status", func(r *PutConfigRequest) { delete(r.StatusSteps, format.BoardDeferred) }},
		{"blank step", func(r *PutConfigRequest) { r.StatusSteps[format.BoardQueued] = " " }},
		{"hidden status mapped", func(r *PutConfigRequest) { r.StatusSteps[format.BoardHidden] = "q" }},
		{"unknown status key", func(r *PutConfigRequest) { r.StatusSteps["someday"] = "q" }},
		{"absolute directory", func(r *PutConfigRequest) { r.Directories = []string{"/etc"} }},
		{"parent directory", func(r *PutConfigRequest) { r.Directories = []string{"docs/../../x"} }},
		{"blank directory", func(r *PutConfigRequest) { r.Directories = []string{"  "} }},
		{"no directories", func(r *PutConfigRequest) { r.Directories = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			req := validRequest()
			tc.mutate(req)

			_, err := svc.PutConfig(context.Background(), testWorkspace, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidConfig)
			stored, getErr := svc.GetConfig(context.Background(), testWorkspace)
			require.NoError(t, getErr)
			assert.Nil(t, stored, "a rejected request must not be stored")
		})
	}
}

func TestPutConfig_CleansAndDeduplicatesDirectories(t *testing.T) {
	svc, _ := newTestService(t)
	req := validRequest()
	req.Directories = []string{"docs/plans", "./docs/plans/", "docs/superpowers/plans"}

	saved, err := svc.PutConfig(context.Background(), testWorkspace, req)

	require.NoError(t, err)
	assert.Equal(t, []string{"docs/plans", "docs/superpowers/plans"}, saved.Directories)
}

func TestGetConfig_AbsentReturnsNil(t *testing.T) {
	svc, _ := newTestService(t)
	got, err := svc.GetConfig(context.Background(), testWorkspace)
	require.NoError(t, err)
	assert.Nil(t, got)
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestService_UnscopedContextIsAllowed(t *testing.T) {
	svc, _ := newTestService(t)
	svc.SetWorkspaceAuthorizer(func(ctx context.Context, _ string) error {
		if ctx.Value(scopedKey{}) == nil {
			return nil
		}
		return repoerrors.ErrWorkspaceNotFound
	})

	_, err := svc.PutConfig(context.Background(), testWorkspace, validRequest())
	require.NoError(t, err)
}

type scopedKey struct{}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestService_ForeignWorkspaceIsNotFound(t *testing.T) {
	svc, fake := newTestService(t)
	svc.SetWorkspaceAuthorizer(func(_ context.Context, _ string) error { return repoerrors.ErrWorkspaceNotFound })
	ctx := context.Background()

	_, err := svc.GetConfig(ctx, testWorkspace)
	assert.ErrorIs(t, err, repoerrors.ErrWorkspaceNotFound)
	_, err = svc.PutConfig(ctx, testWorkspace, validRequest())
	assert.ErrorIs(t, err, repoerrors.ErrWorkspaceNotFound)
	_, err = svc.CreateBoard(ctx, testWorkspace)
	assert.ErrorIs(t, err, repoerrors.ErrWorkspaceNotFound)
	assert.Empty(t, fake.created, "a denied caller must not create a workflow")
}

// @covers AC-TASKS-PLAN-FILES-005.2
func TestCreateBoard_CreatesWorkflowFromTemplateWithMapping(t *testing.T) {
	svc, fake := newTestService(t)

	result, err := svc.CreateBoard(context.Background(), testWorkspace)

	require.NoError(t, err)
	require.Len(t, fake.created, 1)
	assert.Equal(t, testWorkspace, fake.created[0].WorkspaceID)
	require.NotNil(t, fake.created[0].WorkflowTemplateID)
	assert.Equal(t, PlansTemplateID, *fake.created[0].WorkflowTemplateID)
	assert.Equal(t, "created-1", result.WorkflowID)
	assert.Len(t, result.StatusSteps, len(VisibleStatuses()))
	assert.Equal(t, "created-1-queue", result.StatusSteps[format.BoardQueued])
	assert.Equal(t, "created-1-done", result.StatusSteps[format.BoardDone])

	// The mapping the board returns must satisfy PUT validation as is.
	_, err = svc.PutConfig(context.Background(), testWorkspace, &PutConfigRequest{
		Enabled: true, WorkflowID: result.WorkflowID, StatusSteps: result.StatusSteps,
		Directories: DefaultDirectories(),
	})
	require.NoError(t, err)
}

func TestCreateBoard_RemovesWorkflowWhenStepsAreMissing(t *testing.T) {
	svc, fake := newTestService(t)
	fake.dropSteps = true

	_, err := svc.CreateBoard(context.Background(), testWorkspace)

	require.Error(t, err)
	assert.Equal(t, []string{"created-1"}, fake.deleted, "an unusable board must not be left behind")
}
