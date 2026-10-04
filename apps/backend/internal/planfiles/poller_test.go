package planfiles

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kandev/kandev/internal/common/logger"
)

type countingSyncer struct{ calls atomic.Int32 }

func (c *countingSyncer) SyncDueWorkspaces(context.Context) { c.calls.Add(1) }

func TestPoller_StartStopAreIdempotent(t *testing.T) {
	p := NewPoller(&countingSyncer{}, logger.Default())

	p.Start(context.Background())
	p.Start(context.Background())
	p.Stop()
	p.Stop()
}

// @covers AC-TASKS-PLAN-FILES-002.2
func TestPoller_SyncsEveryMinuteAndStopsOnStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		syncer := &countingSyncer{}
		p := NewPoller(syncer, logger.Default())
		p.Start(context.Background())
		t.Cleanup(p.Stop)

		synctest.Wait()
		assert.Zero(t, syncer.calls.Load(), "no pass before the first tick")

		time.Sleep(PollInterval + time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1), syncer.calls.Load())

		time.Sleep(PollInterval)
		synctest.Wait()
		assert.Equal(t, int32(2), syncer.calls.Load())

		p.Stop()
		time.Sleep(3 * PollInterval)
		synctest.Wait()
		assert.Equal(t, int32(2), syncer.calls.Load(), "a stopped poller does not tick")
	})
}

func TestPoller_PollIntervalStaysUnderTheNinetySecondBound(t *testing.T) {
	assert.Less(t, PollInterval, 90*time.Second)
}
