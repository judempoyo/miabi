// SPDX-FileCopyrightText: 2026 Jonas Kaninda
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker

import (
	"context"
	"testing"
	"time"

	"github.com/miabi-io/miabi/internal/models"
	"github.com/miabi-io/miabi/internal/services/analytics"
	"github.com/miabi-io/miabi/internal/storage/repositories"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// unreachableRedis points at a closed port, so every command fails fast without
// needing a server.
func unreachableRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 100 * time.Millisecond,
		MaxRetries:  -1,
	})
}

// The stop function is what shutdown hangs its ordering on: it has to return
// once the consumer is done, promptly, and survive being called twice.
func TestAnalyticsConsumerStartStop(t *testing.T) {
	c := NewAnalyticsConsumer(
		unreachableRedis(), nil, nil,
		"goma:analytics", "test-consumer",
		time.Second, nil, nil,
	)

	stop := c.Start(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		stop()
		stop() // idempotent: a second call must not block or panic
	}()

	select {
	case <-done:
	case <-time.After(analyticsStopTimeout + 2*time.Second):
		t.Fatal("stop() did not return; shutdown would hang")
	}
}

// Cancelling the context the consumer was started with must also bring it down,
// so a caller that cancels a parent context and then calls stop() just waits.
func TestAnalyticsConsumerStopsWithContext(t *testing.T) {
	c := NewAnalyticsConsumer(
		unreachableRedis(), nil, nil,
		"goma:analytics", "test-consumer",
		time.Second, nil, nil,
	)

	ctx, cancel := context.WithCancel(context.Background())
	stop := c.Start(ctx)
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		stop()
	}()
	select {
	case <-done:
	case <-time.After(analyticsStopTimeout + 2*time.Second):
		t.Fatal("stop() did not return after the parent context was cancelled")
	}
}

// The point of waiting on shutdown: buckets live in memory until their minute
// closes, so cancelling has to persist what is open rather than drop it.
func TestAnalyticsFlushLoopPersistsOnCancel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AnalyticsRollup{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := repositories.NewAnalyticsRepository(db)

	c := NewAnalyticsConsumer(
		unreachableRedis(), nil, store,
		"goma:analytics", "test-consumer",
		time.Hour, nil, nil, // flush ticker far away: only the final drain can persist
	)

	// An event old enough to be outside the bucket grace, so the final flush
	// closes its bucket.
	bucket := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Minute)
	c.agg.Ingest(&analytics.Event{
		Ts: bucket.UnixMilli(), Route: "mb-ws9-shop", Method: "GET",
		Status: 200, Path: "/", VID: "v1",
	}, 9, 3)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.flushLoop(ctx)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("flushLoop did not return after cancel")
	}

	rows, err := store.Range(9, nil, bucket.Add(-time.Minute), bucket.Add(time.Minute))
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	if len(rows) != 1 || rows[0].Requests != 1 {
		t.Fatalf("open bucket not persisted on shutdown: %+v", rows)
	}
}

// A nil tracker is replaced with a default, so a caller that doesn't care about
// the live window still gets a working consumer rather than a nil dereference.
func TestAnalyticsConsumerDefaultsLiveTracker(t *testing.T) {
	c := NewAnalyticsConsumer(
		unreachableRedis(), nil, nil,
		"goma:analytics", "test-consumer",
		0, nil, nil,
	)
	if c.live == nil {
		t.Fatal("live tracker not defaulted")
	}
	if c.flushEvery <= 0 {
		t.Fatalf("flushEvery = %v, want a positive default", c.flushEvery)
	}
}
