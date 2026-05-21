package feishu

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserStore_NewUserStore(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create store
	store, err := NewUserStore(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, 0, store.Count())
}

func TestUserStore_GetSet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	// Test Get on empty store
	_, ok := store.GetBySessionID("session1")
	assert.False(t, ok)

	_, ok = store.GetOpenID("session1")
	assert.False(t, ok)

	// Test Set
	userInfo := &UserInfo{
		SessionID: "session1",
		OpenID:    "ou_test123",
		UserID:    "user123",
		IsGroup:   false,
		ChatID:    "chat123",
	}
	err = store.Set(userInfo)
	assert.NoError(t, err)

	// Test Get after Set
	retrieved, ok := store.GetBySessionID("session1")
	assert.True(t, ok)
	assert.Equal(t, "ou_test123", retrieved.OpenID)
	assert.Equal(t, "user123", retrieved.UserID)
	assert.Equal(t, "session1", retrieved.SessionID)
	assert.False(t, retrieved.IsGroup)

	openID, ok := store.GetOpenID("session1")
	assert.True(t, ok)
	assert.Equal(t, "ou_test123", openID)
}

func TestUserStore_Update(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	// Initial set
	userInfo := &UserInfo{
		SessionID: "session1",
		OpenID:    "ou_test123",
		IsGroup:   false,
	}
	err = store.Set(userInfo)
	require.NoError(t, err)

	// Update
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	userInfo.Name = "Updated Name"
	err = store.Set(userInfo)
	require.NoError(t, err)

	// Verify update
	retrieved, ok := store.GetBySessionID("session1")
	assert.True(t, ok)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "ou_test123", retrieved.OpenID) // Preserved
	assert.NotEqual(t, time.Time{}, retrieved.LastSeenAt)
	assert.NotEqual(t, retrieved.FirstSeenAt, retrieved.LastSeenAt)
}

func TestUserStore_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	// Add user
	userInfo := &UserInfo{
		SessionID: "session1",
		OpenID:    "ou_test123",
		IsGroup:   false,
	}
	err = store.Set(userInfo)
	require.NoError(t, err)

	// Delete
	err = store.Delete("session1")
	assert.NoError(t, err)

	// Verify deleted
	_, ok := store.GetBySessionID("session1")
	assert.False(t, ok)
}

func TestUserStore_GetAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	// Add multiple users
	users := []*UserInfo{
		{SessionID: "session1", OpenID: "ou_1", IsGroup: false},
		{SessionID: "session2", OpenID: "ou_2", IsGroup: false},
		{SessionID: "session3", OpenID: "ou_3", IsGroup: true},
	}
	for _, u := range users {
		err = store.Set(u)
		require.NoError(t, err)
	}

	// GetAll
	all := store.GetAll()
	assert.Len(t, all, 3)

	// GetAllSessions
	sessions := store.GetAllSessions()
	assert.Len(t, sessions, 3)
	assert.Contains(t, sessions, "session1")
	assert.Contains(t, sessions, "session2")
	assert.Contains(t, sessions, "session3")
}

func TestUserStore_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create store and add data
	store1, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	userInfo := &UserInfo{
		SessionID: "session1",
		OpenID:    "ou_test123",
		UserID:    "user123",
		IsGroup:   false,
	}
	err = store1.Set(userInfo)
	require.NoError(t, err)

	// Create new store instance (simulates restart)
	store2, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	// Verify data persisted
	retrieved, ok := store2.GetBySessionID("session1")
	assert.True(t, ok)
	assert.Equal(t, "ou_test123", retrieved.OpenID)
	assert.Equal(t, "user123", retrieved.UserID)
}

func TestUserStore_ConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "userstore_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	store, err := NewUserStore(tmpDir)
	require.NoError(t, err)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			userInfo := &UserInfo{
				SessionID: "session" + string(rune('0'+id)),
				OpenID:    "ou_" + string(rune('0'+id)),
				IsGroup:   false,
			}
			store.Set(userInfo)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all writes succeeded
	assert.Equal(t, 10, store.Count())
}
