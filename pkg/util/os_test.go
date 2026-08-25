package util

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUUID_ReturnsAParsableV4UUID(t *testing.T) {
	t.Parallel()

	got := NewUUID()
	parsed, err := uuid.Parse(got)
	if err != nil {
		t.Fatalf("NewUUID() = %q, not a parsable UUID: %v", got, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("NewUUID() version = %d; want 4", parsed.Version())
	}
}

func TestNewUUID_IsNotConstant(t *testing.T) {
	t.Parallel()

	first := NewUUID()
	second := NewUUID()
	if first == second {
		t.Fatalf("two consecutive calls to NewUUID() both returned %q", first)
	}
}
