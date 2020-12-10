package observe

import (
	"context"
	"time"
)

// Poller runs function periodically until exit condition is met
type Poller struct {
	// Interval configures the time interval before reattempting to invoke function
	Interval *time.Duration
	// Timeout specifies the upperbound we will attempt to get a result which meets exit condition
	Timeout *time.Duration
}

// Run runs function until exit condition is met, according to poller settings
func (p *Poller) Run(ctx context.Context, fn func(context.Context) error, exitCond func() bool) error {
	if p.Timeout != nil {
		ctx, _ = context.WithTimeout(ctx, *p.Timeout)
	}

	for {
		if err := fn(ctx); err != nil {
			return err
		}

		if p.Interval == nil || exitCond() {
			break
		}

		select {
		case <-time.After(*p.Interval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
