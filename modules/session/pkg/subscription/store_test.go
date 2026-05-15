package subscription

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscription_NewSubscription(t *testing.T) {
	config := map[string]interface{}{
		"message": "Daily report",
	}
	sub := NewSubscription("", "session1", "daily_report", "0 9 * * * *", config)

	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, "session1", sub.SessionID)
	assert.Equal(t, "daily_report", sub.EventType)
	assert.Equal(t, "0 9 * * * *", sub.CronExpr)
	assert.Equal(t, config, sub.Config)
	assert.True(t, sub.IsActive)
	assert.True(t, sub.LastTrigger.IsZero()) // LastTrigger is zero initially
	assert.False(t, sub.CreatedAt.IsZero())
	assert.False(t, sub.UpdatedAt.IsZero())
}

func TestSubscription_Trigger(t *testing.T) {
	sub := NewSubscription("", "session1", "daily_report", "0 9 * * * *", nil)

	initialTrigger := sub.LastTrigger
	time.Sleep(10 * time.Millisecond)

	sub.Trigger()

	assert.True(t, sub.LastTrigger.After(initialTrigger))
	assert.True(t, sub.UpdatedAt.After(sub.CreatedAt))
}

func TestSubscription_EnableDisable(t *testing.T) {
	sub := NewSubscription("", "session1", "daily_report", "0 9 * * *", nil)

	// Initially active
	assert.True(t, sub.IsActive)

	// Disable
	sub.Disable()
	assert.False(t, sub.IsActive)
	assert.True(t, sub.UpdatedAt.After(sub.CreatedAt))

	// Enable
	sub.Enable()
	assert.True(t, sub.IsActive)
}

func TestSubscriptionStore_NewStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, 0, store.Count())
}

func TestSubscriptionStore_CreateGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create
	sub := NewSubscription("", "session1", "daily_report", "0 9 * * *", nil)
	err = store.Create(sub)
	assert.NoError(t, err)

	// Get
	retrieved, ok := store.Get(sub.ID)
	assert.True(t, ok)
	assert.Equal(t, sub.ID, retrieved.ID)
	assert.Equal(t, "session1", retrieved.SessionID)
	assert.Equal(t, "daily_report", retrieved.EventType)
}

func TestSubscriptionStore_Update(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create
	sub := NewSubscription("", "session1", "daily_report", "0 9 * * * *", nil)
	err = store.Create(sub)
	require.NoError(t, err)

	// Update
	sub.Disable()
	if sub.Config == nil {
		sub.Config = make(map[string]interface{})
	}
	sub.Config["updated"] = true
	err = store.Update(sub)
	assert.NoError(t, err)

	// Verify
	retrieved, ok := store.Get(sub.ID)
	assert.True(t, ok)
	assert.False(t, retrieved.IsActive)
	assert.Equal(t, true, retrieved.Config["updated"])
}

func TestSubscriptionStore_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create
	sub := NewSubscription("", "session1", "daily_report", "0 9 * * * *", nil)
	err = store.Create(sub)
	require.NoError(t, err)

	// Delete
	err = store.Delete(sub.ID)
	assert.NoError(t, err)

	// Verify deleted
	_, ok := store.Get(sub.ID)
	assert.False(t, ok)

	// Delete non-existent
	err = store.Delete("nonexistent")
	assert.Error(t, err)
}

func TestSubscriptionStore_GetBySessionID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create multiple subscriptions for same session
	subs := []*Subscription{
		NewSubscription("", "session1", "daily_report", "0 9 * * * *", nil),
		NewSubscription("", "session1", "weekly_report", "0 10 * * 1 *", nil),
		NewSubscription("", "session2", "daily_report", "0 8 * * * *", nil),
	}
	for _, sub := range subs {
		err = store.Create(sub)
		require.NoError(t, err)
	}

	// GetBySessionID
	session1Subs := store.GetBySessionID("session1")
	assert.Len(t, session1Subs, 2)

	session2Subs := store.GetBySessionID("session2")
	assert.Len(t, session2Subs, 1)

	session3Subs := store.GetBySessionID("session3")
	assert.Nil(t, session3Subs)
}

func TestSubscriptionStore_GetAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create subscriptions
	for i := 0; i < 5; i++ {
		sub := NewSubscription("", "session"+string(rune('0'+i)), "daily_report", "0 9 * * * *", nil)
		err = store.Create(sub)
		require.NoError(t, err)
	}

	// GetAll
	all := store.GetAll()
	assert.Len(t, all, 5)
}

func TestSubscriptionStore_GetActive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create subscriptions
	subs := []*Subscription{
		NewSubscription("", "session1", "daily_report", "0 9 * * * *", nil),
		NewSubscription("", "session2", "daily_report", "0 10 * * * *", nil),
		NewSubscription("", "session3", "daily_report", "0 11 * * * *", nil),
	}
	for _, sub := range subs {
		err = store.Create(sub)
		require.NoError(t, err)
	}

	// Disable one
	subs[1].Disable()
	err = store.Update(subs[1])
	require.NoError(t, err)

	// GetActive
	active := store.GetActive()
	assert.Len(t, active, 2)
}

func TestSubscriptionStore_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "substore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create store and add data
	store1, err := NewStore(tmpDir)
	require.NoError(t, err)

	sub := NewSubscription("", "session1", "daily_report", "0 9 * * * *", nil)
	err = store1.Create(sub)
	require.NoError(t, err)

	// Create new store instance (simulates restart)
	store2, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Verify data persisted
	retrieved, ok := store2.Get(sub.ID)
	assert.True(t, ok)
	assert.Equal(t, "session1", retrieved.SessionID)
	assert.Equal(t, "daily_report", retrieved.EventType)
}
