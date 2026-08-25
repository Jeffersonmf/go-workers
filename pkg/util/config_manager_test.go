package util

import "testing"

func TestReadParameter_ReadsARealEnvironmentVariable(t *testing.T) {
	t.Setenv("GO_WORKERS_TEST_PARAM", "hello")

	got := ReadParameter("GO_WORKERS_TEST_PARAM")
	if got != "hello" {
		t.Fatalf("ReadParameter(GO_WORKERS_TEST_PARAM) = %q; want %q", got, "hello")
	}
}

func TestReadParameter_UnsetVariableReturnsEmptyString(t *testing.T) {
	got := ReadParameter("GO_WORKERS_TEST_PARAM_DEFINITELY_UNSET")
	if got != "" {
		t.Fatalf("ReadParameter(unset) = %q; want \"\"", got)
	}
}
