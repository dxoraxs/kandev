package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/notifications/models"
	"github.com/kandev/kandev/internal/notifications/providers"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

type errProvider struct{}

func (errProvider) Available() bool                       { return true }
func (errProvider) Validate(map[string]interface{}) error { return nil }
func (errProvider) Send(context.Context, providers.Message) error {
	return errors.New("provider down")
}

func planDateService(t *testing.T, repo *notificationTestRepository) *Service {
	t.Helper()
	log, err := logger.NewFromZap(zap.NewNop())
	require.NoError(t, err)
	svc := NewService(repo, notificationTestTaskGetter{workspace: &taskmodels.Workspace{ID: "ws-1", OwnerID: "user-1"}}, nil, log, nil)
	return svc
}

func TestPlanDateReachedIsAnAvailableAndValidEvent(t *testing.T) {
	svc := planDateService(t, &notificationTestRepository{})

	assert.Contains(t, svc.AvailableEvents(), EventPlanFileDateReached)
	assert.Equal(t, "plan_file.date_reached", EventPlanFileDateReached)
	assert.NoError(t, svc.validateEvents([]string{EventPlanFileDateReached}))
	assert.Error(t, svc.validateEvents([]string{"plan_file.unknown"}))
}

func TestHandlePlanDateReachedDeliversToSubscribedEnabledProvidersOnly(t *testing.T) {
	repo := &notificationTestRepository{
		providers: []*models.Provider{
			{ID: "local", UserID: "user-1", Type: models.ProviderTypeLocal, Enabled: true},
			{ID: "apprise", UserID: "user-1", Type: models.ProviderTypeApprise, Enabled: true},
			{ID: "system", UserID: "user-1", Type: models.ProviderTypeSystem, Enabled: false},
		},
		subscriptions: map[string][]*models.Subscription{
			"local":   {{ProviderID: "local", EventType: EventPlanFileDateReached, Enabled: true}},
			"apprise": {{ProviderID: "apprise", EventType: EventSystemUpdateAvailable, Enabled: true}},
			"system":  {{ProviderID: "system", EventType: EventPlanFileDateReached, Enabled: true}},
		},
	}
	svc := planDateService(t, repo)
	local, apprise, system := &captureProvider{}, &captureProvider{}, &captureProvider{}
	svc.providers[models.ProviderTypeLocal] = local
	svc.providers[models.ProviderTypeApprise] = apprise
	svc.providers[models.ProviderTypeSystem] = system

	err := svc.HandlePlanDateReached(context.Background(), "ws-1", "Ship the launch plan", "docs/plans/launch.md")

	require.NoError(t, err)
	require.Len(t, local.messages, 1)
	message := local.messages[0]
	assert.Equal(t, EventPlanFileDateReached, message.EventType)
	assert.Equal(t, "user-1", message.UserID)
	assert.Contains(t, message.Body, "Ship the launch plan")
	assert.NotEmpty(t, message.Title)
	assert.Empty(t, apprise.messages, "a provider not subscribed to the event receives nothing")
	assert.Empty(t, system.messages, "a disabled provider receives nothing")
}

func TestHandlePlanDateReachedReportsAFailedDeliveryAndStillReachesTheOthers(t *testing.T) {
	repo := &notificationTestRepository{
		providers: []*models.Provider{
			{ID: "local", UserID: "user-1", Type: models.ProviderTypeLocal, Enabled: true},
			{ID: "apprise", UserID: "user-1", Type: models.ProviderTypeApprise, Enabled: true},
		},
		subscriptions: map[string][]*models.Subscription{
			"local":   {{ProviderID: "local", EventType: EventPlanFileDateReached, Enabled: true}},
			"apprise": {{ProviderID: "apprise", EventType: EventPlanFileDateReached, Enabled: true}},
		},
	}
	svc := planDateService(t, repo)
	local := &captureProvider{}
	svc.providers[models.ProviderTypeLocal] = local
	svc.providers[models.ProviderTypeApprise] = errProvider{}

	err := svc.NotifyPlanDateReached(context.Background(), "ws-1", "Plan", "docs/plans/p.md")

	require.Error(t, err)
	assert.Len(t, local.messages, 1)
}
