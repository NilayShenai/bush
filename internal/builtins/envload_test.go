package builtins

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEnvload(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	envContent := `
# Sample environment file
APP_NAME=BushShell
APP_PORT=3000
API_KEY=super_secret_token_12345
export DB_HOST="localhost:5432" # inline comment
QUOTED_VAR='hello world'
`
	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test env: %v", err)
	}

	var out bytes.Buffer
	ctx := &ShellContext{
		Stdout: &out,
		Stderr: &out,
	}

	code := builtinEnvload([]string{"envload", envPath}, ctx)
	if code != 0 {
		t.Fatalf("expected envload exit code 0, got %d", code)
	}

	if os.Getenv("APP_NAME") != "BushShell" {
		t.Errorf("expected APP_NAME=BushShell, got '%s'", os.Getenv("APP_NAME"))
	}
	if os.Getenv("DB_HOST") != "localhost:5432" {
		t.Errorf("expected DB_HOST=localhost:5432, got '%s'", os.Getenv("DB_HOST"))
	}
	if os.Getenv("QUOTED_VAR") != "hello world" {
		t.Errorf("expected QUOTED_VAR='hello world', got '%s'", os.Getenv("QUOTED_VAR"))
	}
}
