package observe

import (
	"context"
	"testing"
	"time"
)

func timeDurationPtr(t time.Duration) *time.Duration {
	return &t
}

func TestPoller(t *testing.T) {

	testcases := []struct {
		Name           string // test description
		ExitAfter      int    // number of times exitCondition called before returning true
		Interval       *time.Duration
		Timeout        *time.Duration
		ExpectError    bool
		ExpectNumCalls int // number of times poller we expect invoked function
	}{
		{
			Name:           "Default poller executes function once",
			Interval:       nil,
			Timeout:        nil,
			ExpectNumCalls: 1,
		},
		{
			Name:           "Trigger timeout",
			ExitAfter:      3,
			Interval:       timeDurationPtr(10 * time.Millisecond),
			Timeout:        timeDurationPtr(15 * time.Millisecond),
			ExpectError:    true,
			ExpectNumCalls: 2,
		},
		{
			Name:           "Run to completion",
			ExitAfter:      3,
			ExpectNumCalls: 3,
			Interval:       timeDurationPtr(10 * time.Millisecond),
		},
	}

	for _, tt := range testcases {
		t.Run(tt.Name, func(t *testing.T) {
			p := &Poller{
				Interval: tt.Interval,
				Timeout:  tt.Timeout,
			}

			var numCalls int

			err := p.Run(context.Background(), func(ctx context.Context) error {
				numCalls++
				return nil
			}, func() bool {
				return numCalls >= tt.ExitAfter
			})

			if err != nil && !tt.ExpectError {
				t.Fatalf("unexpected error: %s", err)
			}

			if tt.ExpectError && err == nil {
				t.Fatalf("expected error %d", numCalls)
			}

			if numCalls != tt.ExpectNumCalls {
				t.Fatalf("expected %d function calls, got %d", tt.ExpectNumCalls, numCalls)
			}
		})
	}
}
