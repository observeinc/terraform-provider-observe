package client

import (
	"context"
	"fmt"
	"strconv"
	"sync"
)

// datasetLabelCache holds dataset labels for one Client, and so one
// endpoint, customer and principal. Absent IDs and failures are not cached.
type datasetLabelCache struct {
	mu       sync.Mutex
	labels   map[int64]string
	inflight map[int64]*datasetLabelFetch
}

// datasetLabelFetch is one in-flight lookup; its result fields are set before
// done is closed.
type datasetLabelFetch struct {
	done     chan struct{}
	label    string
	found    bool
	err      error
	canceled bool // owner's context ended; waiters retry
	stale    bool // invalidated while in flight; result is not cached
}

// LookupDatasetLabels returns id→label for the given decimal dataset IDs that
// exist and are visible. Absent or hidden IDs are omitted. Empty input makes no
// request. Invalid IDs return an error.
func (c *Client) LookupDatasetLabels(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	parsed := make([]int64, len(ids))
	pending := make(map[int64]struct{}, len(ids))
	for i, id := range ids {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid dataset id %q", id)
		}
		parsed[i] = n
		pending[n] = struct{}{}
	}

	labels := make(map[int64]string, len(pending))
	for len(pending) > 0 {
		var err error
		if pending, err = c.lookupDatasetLabelsOnce(ctx, pending, labels); err != nil {
			return nil, err
		}
	}

	// Key by the caller's spelling, which may be non-canonical (e.g. "007").
	result := make(map[string]string, len(labels))
	for i, id := range ids {
		if label, ok := labels[parsed[i]]; ok {
			result[id] = label
		}
	}
	return result, nil
}

// lookupDatasetLabelsOnce resolves pending into labels from the cache, its own
// fetch, or fetches owned by concurrent callers. It returns the IDs whose
// fetch owner was canceled, which the caller must retry.
func (c *Client) lookupDatasetLabelsOnce(ctx context.Context, pending map[int64]struct{}, labels map[int64]string) (map[int64]struct{}, error) {
	cache := &c.datasetLabels
	owned := map[int64]*datasetLabelFetch{}
	waits := map[int64]*datasetLabelFetch{}

	cache.mu.Lock()
	if cache.labels == nil {
		cache.labels = map[int64]string{}
		cache.inflight = map[int64]*datasetLabelFetch{}
	}
	for id := range pending {
		if label, ok := cache.labels[id]; ok {
			labels[id] = label
		} else if f, ok := cache.inflight[id]; ok {
			waits[id] = f
		} else {
			f := &datasetLabelFetch{done: make(chan struct{})}
			cache.inflight[id] = f
			owned[id] = f
		}
	}
	cache.mu.Unlock()

	if len(owned) > 0 {
		ids := make([]int64, 0, len(owned))
		for id := range owned {
			ids = append(ids, id)
		}
		fetched, err := c.Rest.LookupDatasetLabels(ctx, ids)
		canceled := err != nil && ctx.Err() != nil

		cache.mu.Lock()
		for id, f := range owned {
			f.label, f.found = fetched[id]
			f.err, f.canceled = err, canceled
			if cache.inflight[id] == f {
				delete(cache.inflight, id)
			}
			if err == nil && f.found && !f.stale {
				cache.labels[id] = f.label
			}
			close(f.done)
		}
		cache.mu.Unlock()

		if err != nil {
			return nil, err
		}
		for id, label := range fetched {
			labels[id] = label
		}
	}

	retry := map[int64]struct{}{}
	for id, f := range waits {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-f.done:
		}
		switch {
		case f.canceled:
			retry[id] = struct{}{}
		case f.err != nil:
			return nil, f.err
		case f.found:
			labels[id] = f.label
		}
	}
	return retry, nil
}

// invalidateDatasetLabel drops id's cached label and detaches any in-flight
// fetch for it. Unparseable IDs are ignored.
func (c *Client) invalidateDatasetLabel(id string) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return
	}
	cache := &c.datasetLabels
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.labels, n)
	if f, ok := cache.inflight[n]; ok {
		f.stale = true
		delete(cache.inflight, n)
	}
}

// invalidateSavedDatasetLabel invalidates a saved dataset's input and result
// IDs, which differ when the server assigns the ID.
func (c *Client) invalidateSavedDatasetLabel(inputID *string, resultID string) {
	if inputID != nil {
		c.invalidateDatasetLabel(*inputID)
	}
	c.invalidateDatasetLabel(resultID)
}

// clearDatasetLabels invalidates every cached and in-flight label.
func (c *Client) clearDatasetLabels() {
	cache := &c.datasetLabels
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for _, f := range cache.inflight {
		f.stale = true
	}
	cache.labels = nil
	cache.inflight = nil
}
