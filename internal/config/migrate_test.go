package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateLegacyPathsMovesDirsWhenNewMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldConfig := filepath.Join(home, ".config", "investments-healthcheck")
	if err := os.MkdirAll(oldConfig, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldConfig, "config.yaml"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	MigrateLegacyPaths(&buf)

	newConfig := filepath.Join(home, ".config", "fundpulse")
	if _, err := os.Stat(filepath.Join(newConfig, "config.yaml")); err != nil {
		t.Errorf("new path missing after migration: %v", err)
	}
	if _, err := os.Stat(oldConfig); !os.IsNotExist(err) {
		t.Errorf("old path should be gone after migration; got err=%v", err)
	}
	if !strings.Contains(buf.String(), "Migrated") {
		t.Errorf("expected notice, got %q", buf.String())
	}
}

func TestMigrateLegacyPathsSkipsWhenNewExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldConfig := filepath.Join(home, ".config", "investments-healthcheck")
	newConfig := filepath.Join(home, ".config", "fundpulse")
	for _, d := range []string{oldConfig, newConfig} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldConfig, "config.yaml"), []byte("OLD"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newConfig, "config.yaml"), []byte("NEW"), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	MigrateLegacyPaths(&buf)

	got, err := os.ReadFile(filepath.Join(newConfig, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "NEW" {
		t.Errorf("new file overwritten by migration; content=%q", got)
	}
}

func TestMigrateLegacyPathsNoOpWhenNothingExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var buf bytes.Buffer
	MigrateLegacyPaths(&buf)
	if buf.Len() != 0 {
		t.Errorf("expected no output when nothing to migrate; got %q", buf.String())
	}
}
