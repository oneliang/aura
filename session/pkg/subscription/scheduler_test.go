package subscription

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oneliang/aura/shared/pkg/logger"
)

func TestScheduler_NewScheduler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	var triggerCount int
	var mu sync.Mutex
	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		mu.Lock()
		triggerCount++
		mu.Unlock()
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)
	assert.NotNil(t, scheduler)
}

func TestScheduler_AddRemoveSubscription(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	var mu sync.Mutex
	triggered := make(chan string, 10)
	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		mu.Lock()
		defer mu.Unlock()
		triggered <- sub.ID
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)

	// Add subscription
	sub := NewSubscription("", "session1", "test_event", "0 */1 * * * *", nil) // Every minute for testing (6 fields)
	err = scheduler.AddSubscription(sub)
	assert.NoError(t, err)

	// Verify added
	retrieved, ok := scheduler.GetSubscription(sub.ID)
	assert.True(t, ok)
	assert.Equal(t, sub.ID, retrieved.ID)

	// Remove subscription
	err = scheduler.RemoveSubscription(sub.ID)
	assert.NoError(t, err)

	// Verify removed
	_, ok = scheduler.GetSubscription(sub.ID)
	assert.False(t, ok)
}

func TestScheduler_StartStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)

	// Start
	scheduler.Start()
	assert.True(t, scheduler.running)

	// Stop
	scheduler.Stop()
	assert.False(t, scheduler.running)
}

func TestScheduler_TriggerSubscription(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	var mu sync.Mutex
	var lastTriggered string
	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		mu.Lock()
		lastTriggered = sub.ID
		mu.Unlock()
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)

	// Add subscription
	sub := NewSubscription("", "session1", "test_event", "0 9 * * * *", nil)
	err = scheduler.AddSubscription(sub)
	require.NoError(t, err)

	// Manually trigger
	err = scheduler.TriggerSubscription(sub.ID)
	assert.NoError(t, err)

	// Verify triggered
	mu.Lock()
	triggeredID := lastTriggered
	mu.Unlock()
	assert.Equal(t, sub.ID, triggeredID)

	// Verify LastTrigger updated
	updatedSub, _ := store.Get(sub.ID)
	assert.False(t, updatedSub.LastTrigger.IsZero())
}

func TestScheduler_ListSubscriptions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)

	// Add multiple subscriptions
	for i := 0; i < 3; i++ {
		sub := NewSubscription("", "session"+string(rune('0'+i)), "daily_report", "0 9 * * * *", nil)
		err = scheduler.AddSubscription(sub)
		require.NoError(t, err)
	}

	// List
	all := scheduler.ListSubscriptions()
	assert.Len(t, all, 3)
}

func TestScheduler_InvalidCronExpression(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)

	// Invalid cron expression
	sub := NewSubscription("", "session1", "test_event", "invalid cron", nil)
	err = scheduler.AddSubscription(sub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestScheduler_ConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "scheduler_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	triggerFunc := func(ctx context.Context, sub *Subscription) error {
		return nil
	}

	log := logger.New(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "discard",
	})

	scheduler := NewScheduler(store, triggerFunc, log)
	scheduler.Start()
	defer scheduler.Stop()

	// Concurrent add/remove
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sub := NewSubscription("", "session"+string(rune('0'+id)), "test", "0 9 * * *", nil)
			err := scheduler.AddSubscription(sub)
			if err == nil {
				time.Sleep(time.Millisecond)
				scheduler.RemoveSubscription(sub.ID)
			}
		}(i)
	}

	wg.Wait()
}
