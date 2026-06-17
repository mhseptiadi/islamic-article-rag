package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func TestLoadReadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("HTTP_PORT=9091\nLLM_PROVIDER=google\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPPort != "9091" {
		t.Fatalf("HTTPPort = %q, want 9091", cfg.HTTPPort)
	}
	if cfg.LLMProvider != "google" {
		t.Fatalf("LLMProvider = %q, want google", cfg.LLMProvider)
	}
}

func TestStripSemicolonComments(t *testing.T) {
	input := []byte("; commented\nHTTP_PORT=7070\n")
	out := stripSemicolonComments(input)
	m, err := loadFileEnvFromContent(out)
	if err != nil {
		t.Fatal(err)
	}
	if m["HTTP_PORT"] != "7070" {
		t.Fatalf("HTTP_PORT = %q, want 7070", m["HTTP_PORT"])
	}
}

func loadFileEnvFromContent(data []byte) (map[string]string, error) {
	return godotenv.Unmarshal(string(stripSemicolonComments(data)))
}
