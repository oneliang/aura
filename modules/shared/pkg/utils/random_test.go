package utils

import (
	"testing"
)

func TestRandString(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		s, err := RandString(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(s) != 10 {
			t.Errorf("expected length 10, got %d", len(s))
		}
	})

	t.Run("zero length", func(t *testing.T) {
		s, err := RandString(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("expected empty string, got %q", s)
		}
	})

	t.Run("negative length", func(t *testing.T) {
		s, err := RandString(-1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s != "" {
			t.Errorf("expected empty string, got %q", s)
		}
	})

	t.Run("only alphanumeric characters", func(t *testing.T) {
		s, err := RandString(100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, c := range s {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				t.Errorf("invalid character %q in generated string", c)
			}
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		strings := make(map[string]bool)
		for i := 0; i < 100; i++ {
			s, err := RandString(16)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings[s] {
				t.Errorf("duplicate string generated: %s", s)
			}
			strings[s] = true
		}
	})
}

func TestMustRandString(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		s := MustRandString(10)
		if len(s) != 10 {
			t.Errorf("expected length 10, got %d", len(s))
		}
	})

	t.Run("zero length", func(t *testing.T) {
		s := MustRandString(0)
		if s != "" {
			t.Errorf("expected empty string, got %q", s)
		}
	})
}
