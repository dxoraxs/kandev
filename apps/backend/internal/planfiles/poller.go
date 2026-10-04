package planfiles

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
)

// PollInterval is the sync cadence. With the scan itself taking seconds at
// most, a file change reaches the board within the 90-second bound.
const PollInterval = 60 * time.Second

// dueSyncer runs a pass for every workspace that has plan files enabled.
// Satisfied by *Service.
type dueSyncer interface {
	SyncDueWorkspaces(ctx context.Context)
}

// Poller periodically syncs plan files for every enabled workspace. It owns
// one goroutine with idempotent Start and Stop.
type Poller struct {
	syncer   dueSyncer
	logger   *logger.Logger
	interval time.Duration

	mu      sync.Mutex
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

// NewPoller creates a plan-file poller at the default interval.
func NewPoller(syncer dueSyncer, log *logger.Logger) *Poller {
	return &Poller{
		syncer:   syncer,
		logger:   log.WithFields(zap.String("component", "planfiles-poller")),
		interval: PollInterval,
	}
}

// Start launches the polling loop. Idempotent.
func (p *Poller) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return
	}
	p.started = true
	ctx, p.cancel = context.WithCancel(ctx)
	p.wg.Add(1)
	go p.loop(ctx)
}

// Stop cancels the loop and waits for it to drain. Idempotent. The mutex is
// held through the wait so a concurrent Start cannot register a new loop
// mid-shutdown; the loop never takes the mutex, so this cannot deadlock.
func (p *Poller) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		return
	}
	p.started = false
	p.cancel()
	p.wg.Wait()
}

// loop waits a full interval before the first pass so boot stays quiet.
func (p *Poller) loop(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.syncer.SyncDueWorkspaces(ctx)
		}
	}
}
