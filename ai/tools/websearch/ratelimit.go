package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type searxngRequestGate struct {
	slots       chan struct{}
	mu          sync.Mutex
	retryUntil  time.Time
	retryStatus int
	now         func() time.Time
}

var searxngInstanceGates = struct {
	sync.Mutex
	byEndpoint map[string]*searxngRequestGate
}{byEndpoint: make(map[string]*searxngRequestGate)}

func instanceRequestGate(endpoint string, concurrent int) (*searxngRequestGate, error) {
	searxngInstanceGates.Lock()
	defer searxngInstanceGates.Unlock()
	if gate := searxngInstanceGates.byEndpoint[endpoint]; gate != nil {
		if cap(gate.slots) != concurrent {
			return nil, fmt.Errorf("SearXNG concurrency for this instance is already configured; restart the application to change it")
		}
		return gate, nil
	}
	gate := &searxngRequestGate{slots: make(chan struct{}, concurrent), now: time.Now}
	searxngInstanceGates.byEndpoint[endpoint] = gate
	return gate, nil
}

func requestGateNow(gate *searxngRequestGate) time.Time {
	if gate != nil && gate.now != nil {
		return gate.now()
	}
	return time.Now()
}

func (gate *searxngRequestGate) cooldownError() error {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	now := requestGateNow(gate)
	if now.Before(gate.retryUntil) {
		return rateLimitMessage(gate.retryStatus, gate.retryUntil, now)
	}
	return nil
}

func (gate *searxngRequestGate) acquire(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := gate.cooldownError(); err != nil {
		return nil, err
	}
	select {
	case gate.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := func() { <-gate.slots }
	if err := ctx.Err(); err != nil {
		release()
		return nil, err
	}
	if err := gate.cooldownError(); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func (gate *searxngRequestGate) cooldown(status int, until time.Time) {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if until.After(gate.retryUntil) {
		gate.retryUntil, gate.retryStatus = until, status
	}
}

func retryAfterDeadline(value string, now time.Time) time.Time {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		const maxDurationSeconds = int64((1<<63 - 1) / time.Second)
		return now.Add(time.Duration(min(seconds, maxDurationSeconds)) * time.Second)
	}
	if deadline, err := http.ParseTime(value); err == nil {
		if deadline.Before(now) {
			return now
		}
		return deadline
	}
	return now.Add(30 * time.Second)
}

func rateLimitMessage(status int, deadline, now time.Time) error {
	remaining := max(time.Duration(0), deadline.Sub(now))
	seconds := int64(remaining / time.Second)
	if remaining%time.Second > 0 {
		seconds++
	}
	return fmt.Errorf("SearXNG returned HTTP %d; retry after %d seconds (not before %s). No automatic retry or provider fallback was performed", status, seconds, deadline.UTC().Format(time.RFC3339))
}

type requestGateBody struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (body *requestGateBody) Close() error {
	err := body.ReadCloser.Close()
	body.once.Do(body.release)
	return err
}
