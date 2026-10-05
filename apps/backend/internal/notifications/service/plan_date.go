package service

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

// EventPlanFileDateReached is sent when a plan file wakes up on its date.
const EventPlanFileDateReached = "plan_file.date_reached"

// HandlePlanDateReached notifies the owner of the workspace that a plan file
// reached its date and waits for them. Every enabled provider subscribed to
// the event receives it; a failed delivery does not stop the others and is
// returned joined.
func (s *Service) HandlePlanDateReached(ctx context.Context, workspaceID, title, relPath string) error {
	userID, ok := s.workspaceOwner(ctx, workspaceID)
	if !ok {
		return nil
	}
	providers, subscriptions, err := s.ListProviders(ctx, userID)
	if err != nil {
		return fmt.Errorf("load notification providers: %w", err)
	}
	notifTitle := "Plan date reached"
	body := fmt.Sprintf("%q is waiting for you.", title)
	var failures []error
	for _, provider := range providers {
		if !provider.Enabled || !containsEvent(subscriptions[provider.ID], EventPlanFileDateReached) {
			continue
		}
		if err := s.dispatchGenericNotification(ctx, userID, provider, EventPlanFileDateReached, notifTitle, body); err != nil {
			s.logger.Warn("plan date notification delivery failed",
				zap.String("provider_id", provider.ID), zap.String("rel_path", relPath), zap.Error(err))
			failures = append(failures, fmt.Errorf("provider %s: %w", provider.ID, err))
		}
	}
	return errors.Join(failures...)
}

// NotifyPlanDateReached satisfies planfiles.DateNotifier.
func (s *Service) NotifyPlanDateReached(ctx context.Context, workspaceID, title, relPath string) error {
	return s.HandlePlanDateReached(ctx, workspaceID, title, relPath)
}
