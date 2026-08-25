package util

import (
	"os"
	"testing"
)

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

func TestLoadEnvFile_PopulatesReadParameterFromDotEnv(t *testing.T) {
	restoreEnvFileVars(t)
	t.Chdir(t.TempDir())

	writeDotEnv(t, "FROM_FILE=file-value\n# a comment\n\nQUOTED=\"quoted value\"\n")

	loadEnvFile()

	if got := ReadParameter("FROM_FILE"); got != "file-value" {
		t.Fatalf("ReadParameter(FROM_FILE) = %q; want %q", got, "file-value")
	}
	if got := ReadParameter("QUOTED"); got != "quoted value" {
		t.Fatalf("ReadParameter(QUOTED) = %q; want %q (surrounding quotes stripped)", got, "quoted value")
	}
}

func TestReadParameter_RealEnvVarOverridesDotEnv(t *testing.T) {
	restoreEnvFileVars(t)
	t.Chdir(t.TempDir())

	writeDotEnv(t, "SHARED_KEY=from-file\n")
	loadEnvFile()

	t.Setenv("SHARED_KEY", "from-real-env")

	if got := ReadParameter("SHARED_KEY"); got != "from-real-env" {
		t.Fatalf("ReadParameter(SHARED_KEY) = %q; want the real env var to win over .env", got)
	}
}

func TestLoadEnvFile_MissingFileLeavesNoValuesAndDoesNotPanic(t *testing.T) {
	restoreEnvFileVars(t)
	t.Chdir(t.TempDir())

	loadEnvFile()

	if got := ReadParameter("ANYTHING"); got != "" {
		t.Fatalf("ReadParameter(ANYTHING) = %q; want \"\" with no .env present", got)
	}
}

// restoreEnvFileVars resets the package-level .env cache to empty after
// the test, so a test that loads a temp-dir .env cannot leak values
// into tests that run afterward.
func restoreEnvFileVars(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		envFileMu.Lock()
		envFileVars = map[string]string{}
		envFileMu.Unlock()
	})
}

func writeDotEnv(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(envFilePath, []byte(content), 0o600); err != nil {
		t.Fatalf("could not write .env fixture: %v", err)
	}
}
