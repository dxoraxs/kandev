package planfiles

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

// An absent field keeps the stored value, so a body from an older client does
// not reset the operation settings.
func TestPutConfig_AbsentOperationFieldsKeepStoredValues(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	req := validRequest()
	req.ExecutorSteps = ptr(map[string]string{"extra": "Claude"})
	req.NotesHeading = ptr("My notes")
	req.WakeOnDate = ptr(false)
	req.StaleAfterDays = ptr(14)
	req.IndexFile = ptr("INDEX.md")
	_, err := svc.PutConfig(ctx, testWorkspace, req)
	require.NoError(t, err)

	saved, err := svc.PutConfig(ctx, testWorkspace, validRequest())

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"extra": "Claude"}, saved.ExecutorSteps)
	assert.Equal(t, "My notes", saved.NotesHeading)
	assert.False(t, saved.WakeOnDate)
	assert.Equal(t, 14, saved.StaleAfterDays)
	assert.Equal(t, "INDEX.md", saved.IndexFile)
}

func TestPutConfig_NewConfigAppliesDefaultsForAbsentFields(t *testing.T) {
	svc, _ := newTestService(t)

	saved, err := svc.PutConfig(context.Background(), testWorkspace, validRequest())

	require.NoError(t, err)
	assert.NotNil(t, saved.ExecutorSteps)
	assert.Empty(t, saved.ExecutorSteps)
	assert.Equal(t, "", saved.NotesHeading)
	assert.True(t, saved.WakeOnDate)
	assert.Equal(t, 7, saved.StaleAfterDays)
	assert.Equal(t, "", saved.IndexFile)
}

func TestPutConfig_ExplicitEmptyValuesClearStoredSettings(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	req := validRequest()
	req.ExecutorSteps = ptr(map[string]string{"extra": "Claude"})
	req.IndexFile = ptr("INDEX.md")
	_, err := svc.PutConfig(ctx, testWorkspace, req)
	require.NoError(t, err)

	clear := validRequest()
	clear.ExecutorSteps = ptr(map[string]string{})
	clear.IndexFile = ptr("")
	saved, err := svc.PutConfig(ctx, testWorkspace, clear)

	require.NoError(t, err)
	assert.Empty(t, saved.ExecutorSteps)
	assert.Equal(t, "", saved.IndexFile)
}

func TestPutConfig_TrimsAndStoresExecutorNames(t *testing.T) {
	svc, _ := newTestService(t)
	req := validRequest()
	req.ExecutorSteps = ptr(map[string]string{"extra": "  Claude  "})

	saved, err := svc.PutConfig(context.Background(), testWorkspace, req)

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"extra": "Claude"}, saved.ExecutorSteps)
}

func TestPutConfig_RejectsInvalidOperationSettings(t *testing.T) {
	cases := []struct {
		name   string
		code   string
		mutate func(r *PutConfigRequest)
	}{
		{"mapped status step", "invalid_executor_steps", func(r *PutConfigRequest) {
			r.ExecutorSteps = ptr(map[string]string{"q": "Claude"})
		}},
		{"unknown step", "invalid_executor_steps", func(r *PutConfigRequest) {
			r.ExecutorSteps = ptr(map[string]string{"nope": "Claude"})
		}},
		{"duplicate name", "invalid_executor_steps", func(r *PutConfigRequest) {
			r.ExecutorSteps = ptr(map[string]string{"extra": "Claude", "wo2": "claude"})
		}},
		{"blank name", "invalid_executor_steps", func(r *PutConfigRequest) {
			r.ExecutorSteps = ptr(map[string]string{"extra": "   "})
		}},
		{"long name", "invalid_executor_steps", func(r *PutConfigRequest) {
			r.ExecutorSteps = ptr(map[string]string{"extra": strings.Repeat("a", 41)})
		}},
		{"long heading", "invalid_notes_heading", func(r *PutConfigRequest) {
			r.NotesHeading = ptr(strings.Repeat("h", 101))
		}},
		{"multiline heading", "invalid_notes_heading", func(r *PutConfigRequest) {
			r.NotesHeading = ptr("a\nb")
		}},
		{"negative stale", "invalid_stale_after_days", func(r *PutConfigRequest) { r.StaleAfterDays = ptr(-1) }},
		{"huge stale", "invalid_stale_after_days", func(r *PutConfigRequest) { r.StaleAfterDays = ptr(366) }},
		{"index with path", "invalid_index_file", func(r *PutConfigRequest) { r.IndexFile = ptr("a/INDEX.md") }},
		{"index not markdown", "invalid_index_file", func(r *PutConfigRequest) { r.IndexFile = ptr("INDEX.txt") }},
		{"index hidden", "invalid_index_file", func(r *PutConfigRequest) { r.IndexFile = ptr(".INDEX.md") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, fake := newTestService(t)
			fake.addWorkflow("wf-1", testWorkspace, "q", "ip", "wo", "we", "df", "dn", "extra", "wo2")
			req := validRequest()
			tc.mutate(req)

			_, err := svc.PutConfig(context.Background(), testWorkspace, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidConfig)
			var cfgErr *ConfigError
			require.ErrorAs(t, err, &cfgErr)
			assert.Equal(t, tc.code, cfgErr.Code)
			stored, getErr := svc.GetConfig(context.Background(), testWorkspace)
			require.NoError(t, getErr)
			assert.Nil(t, stored, "a rejected request must not be stored")
		})
	}
}
