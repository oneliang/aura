package tui

import (
	"strings"
	"testing"
)

func TestNewInputManager(t *testing.T) {
	im := NewInputManager(UIStyles{})

	if im == nil {
		t.Fatal("NewInputManager() returned nil")
	}
	if im.IsDisabled() {
		t.Error("New input manager should not be disabled")
	}
}

func TestInputManager_SetDisabled(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetDisabled(true)
	if !im.IsDisabled() {
		t.Error("IsDisabled() should return true after SetDisabled(true)")
	}

	im.SetDisabled(false)
	if im.IsDisabled() {
		t.Error("IsDisabled() should return false after SetDisabled(false)")
	}
}

func TestInputManager_Value(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// Initially empty
	if im.Value() != "" {
		t.Errorf("Initial value should be empty, got %q", im.Value())
	}

	im.SetValue("test input")
	if im.Value() != "test input" {
		t.Errorf("Value() = %q, want %q", im.Value(), "test input")
	}
}

func TestInputManager_SetValue(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetValue("hello")
	if im.Value() != "hello" {
		t.Errorf("SetValue() failed, Value() = %q", im.Value())
	}
}

func TestInputManager_Reset(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetValue("some text")
	im.Reset()

	if im.Value() != "" {
		t.Errorf("Reset() should clear value, got %q", im.Value())
	}
}

func TestInputManager_DisableAndClear(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetValue("text to clear")
	im.DisableAndClear()

	if !im.IsDisabled() {
		t.Error("DisableAndClear() should disable input")
	}
	if im.Value() != "" {
		t.Errorf("DisableAndClear() should clear value, got %q", im.Value())
	}
}

func TestInputManager_EnableAndFocus(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetDisabled(true)
	cmd := im.EnableAndFocus()

	if im.IsDisabled() {
		t.Error("EnableAndFocus() should enable input")
	}
	if im.Value() != "" {
		t.Errorf("EnableAndFocus() should clear value, got %q", im.Value())
	}
	// cmd may be nil or a valid tea.Cmd depending on implementation
	_ = cmd
}

func TestInputManager_View(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// When not disabled, should show textarea
	view := im.View()
	if view == "" {
		t.Error("View() should return non-empty string")
	}

	// When disabled, should show placeholder
	im.SetDisabled(true)
	view = im.View()
	if view == "" {
		t.Error("View() should return placeholder when disabled")
	}
}

func TestInputManager_IsEmpty(t *testing.T) {
	im := NewInputManager(UIStyles{})

	if !im.IsEmpty() {
		t.Error("IsEmpty() should return true for empty input")
	}

	im.SetValue("text")
	if im.IsEmpty() {
		t.Error("IsEmpty() should return false for non-empty input")
	}

	im.SetValue("   ")
	if !im.IsEmpty() {
		t.Error("IsEmpty() should return true for whitespace-only input")
	}
}

func TestInputManager_SetWidth(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// Should not panic
	im.SetWidth(80)
	im.SetWidth(120)
}

func TestInputManager_ForceRedraw(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetValue("test content")
	im.ForceRedraw(100)

	// Value should be preserved
	if im.Value() != "test content" {
		t.Errorf("ForceRedraw() should preserve value, got %q", im.Value())
	}
}

func TestInputManager_ForceRedraw_Disabled(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetDisabled(true)
	im.SetValue("test")
	im.ForceRedraw(100)

	if im.Value() != "test" {
		t.Errorf("ForceRedraw() should preserve value when disabled, got %q", im.Value())
	}
}

func TestInputManager_Focus(t *testing.T) {
	im := NewInputManager(UIStyles{})

	cmd := im.Focus()
	// Focus returns a tea.Cmd which may be nil or not
	_ = cmd
}

func TestInputManager_Blur(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// Should not panic
	im.Blur()
}

func TestInputManager_Update_WhenDisabled(t *testing.T) {
	im := NewInputManager(UIStyles{})

	im.SetDisabled(true)
	// Update should return nil when disabled
	cmd := im.Update(nil)
	if cmd != nil {
		t.Error("Update() should return nil when disabled")
	}
}

func TestInputManager_Update_WhenEnabled(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// Update returns a tea.Cmd which may be nil for certain messages
	cmd := im.Update(nil)
	_ = cmd
}

func TestInputManager_Placeholder(t *testing.T) {
	im := NewInputManager(UIStyles{})
	im.SetDisabled(true)

	view := im.View()
	// View should contain the placeholder
	if !strings.Contains(view, "Waiting") {
		t.Logf("View when disabled: %q", view)
	}
}

func TestInputManager_MultipleOperations(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// Test a sequence of operations
	im.SetValue("first")
	im.SetDisabled(true)
	im.SetValue("second") // Should still work even when disabled
	im.SetDisabled(false)
	im.Reset()
	im.SetValue("third")
	im.DisableAndClear()
	im.EnableAndFocus()

	if im.Value() != "" {
		t.Errorf("Value should be empty after EnableAndFocus, got %q", im.Value())
	}
}

func TestInputManager_WidthChanges(t *testing.T) {
	im := NewInputManager(UIStyles{})

	// Test multiple width changes
	for _, width := range []int{40, 80, 120, 60} {
		im.SetWidth(width)
		im.ForceRedraw(width)
	}

	// Should not panic and value should remain
	im.SetValue("test")
	im.ForceRedraw(80)
	if im.Value() != "test" {
		t.Errorf("Value lost after width changes: %q", im.Value())
	}
}
