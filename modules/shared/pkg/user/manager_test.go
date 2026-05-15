// Package user provides user management for multi-user support.
package user

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oneliang/aura/shared/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestNewManager(t *testing.T) {
	cfg := &config.UsersConfig{
		Default: "user_001",
		Definitions: []config.UserConfig{
			{
				ID:       "user_001",
				Name:     "张三",
				APIToken: "sk_test123",
			},
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if m.defaultID != "user_001" {
		t.Errorf("defaultID = %q, want %q", m.defaultID, "user_001")
	}

	user := m.GetUserByID("user_001")
	if user == nil {
		t.Error("GetUserByID() returned nil")
	}
	if user.Name != "张三" {
		t.Errorf("Name = %q, want %q", user.Name, "张三")
	}
}

func TestManagerCreateUser(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.UsersConfig{
		Default:     "default",
		Definitions: []config.UserConfig{},
	}

	m, err := NewManagerWithBaseDir(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() error = %v", err)
	}

	user, err := m.CreateUser("测试用户")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if user.ID == "" {
		t.Error("CreateUser() returned empty ID")
	}
	if user.Name != "测试用户" {
		t.Errorf("Name = %q, want %q", user.Name, "测试用户")
	}
	if user.APIToken == "" {
		t.Error("CreateUser() returned empty APIToken")
	}

	// Verify user is indexed
	retrieved := m.GetUserByID(user.ID)
	if retrieved == nil {
		t.Error("GetUserByID() returned nil after CreateUser")
	}

	// Verify token is indexed
	byToken := m.GetUserByToken(user.APIToken)
	if byToken == nil {
		t.Error("GetUserByToken() returned nil after CreateUser")
	}
}

func TestManagerDeleteUser(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.UsersConfig{
		Default: "user_001",
		Definitions: []config.UserConfig{
			{
				ID:       "user_001",
				Name:     "张三",
				APIToken: "sk_test123",
			},
		},
	}

	m, err := NewManagerWithBaseDir(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() error = %v", err)
	}

	// Delete the user
	err = m.DeleteUser("user_001")
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}

	// Verify user is removed
	if m.GetUserByID("user_001") != nil {
		t.Error("GetUserByID() should return nil after DeleteUser")
	}
	if m.GetUserByToken("sk_test123") != nil {
		t.Error("GetUserByToken() should return nil after DeleteUser")
	}
}

func TestManagerSwitchUser(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.UsersConfig{
		Default: "user_001",
		Definitions: []config.UserConfig{
			{ID: "user_001", Name: "张三"},
			{ID: "user_002", Name: "李四"},
		},
	}

	m, err := NewManagerWithBaseDir(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() error = %v", err)
	}

	// Switch to user_002
	err = m.SwitchUser("user_002")
	if err != nil {
		t.Fatalf("SwitchUser() error = %v", err)
	}

	// Verify default user changed
	if m.defaultID != "user_002" {
		t.Errorf("defaultID = %q, want %q", m.defaultID, "user_002")
	}

	// Switch to non-existent user should fail
	err = m.SwitchUser("non_existent")
	if err == nil {
		t.Error("SwitchUser() should fail for non-existent user")
	}
}

func TestManagerListUsers(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.UsersConfig{
		Default: "user_001",
		Definitions: []config.UserConfig{
			{ID: "user_001", Name: "张三"},
			{ID: "user_002", Name: "李四"},
		},
	}

	m, err := NewManagerWithBaseDir(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewManagerWithBaseDir() error = %v", err)
	}

	users := m.ListUsers()
	if len(users) != 2 {
		t.Errorf("ListUsers() returned %d users, want 2", len(users))
	}
}

func TestManagerSave(t *testing.T) {
	// Create a temporary home directory
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cfg := &config.UsersConfig{
		Default: "user_001",
		Definitions: []config.UserConfig{
			{ID: "user_001", Name: "张三", APIToken: "sk_test123"},
		},
	}

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Create a new user
	newUser, err := m.CreateUser("新用户")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// Save to disk
	err = m.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	usersFile := filepath.Join(tmpHome, ".aura", "users.yaml")
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		t.Error("users.yaml was not created")
	}

	// Verify we can reload by reading the file directly
	data, err := os.ReadFile(usersFile)
	if err != nil {
		t.Fatalf("Failed to read users.yaml: %v", err)
	}

	var loadedCfg config.UsersConfig
	if err := yaml.Unmarshal(data, &loadedCfg); err != nil {
		t.Fatalf("Failed to parse users.yaml: %v", err)
	}

	// Check if new user is persisted
	found := false
	for _, u := range loadedCfg.Definitions {
		if u.ID == newUser.ID {
			found = true
			if u.Name != newUser.Name {
				t.Errorf("Loaded user name = %q, want %q", u.Name, newUser.Name)
			}
			break
		}
	}
	if !found {
		t.Error("New user was not persisted")
	}
}
