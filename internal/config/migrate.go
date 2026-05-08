package config

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// MigrateLegacyPaths is best-effort: it relocates pre-rename
// data ($HOME/.config/investments-healthcheck etc.) to the new
// fundpulse paths and removes the old launchd plist.
//
// Idempotent — does nothing if the new paths already exist.
// Writes a one-line notice to w when it actually moves something.
func MigrateLegacyPaths(w io.Writer) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}

	dirs := []struct{ old, new string }{
		{
			filepath.Join(home, ".config", "investments-healthcheck"),
			filepath.Join(home, ".config", "fundpulse"),
		},
		{
			filepath.Join(home, "Library", "Application Support", "investments-healthcheck"),
			filepath.Join(home, "Library", "Application Support", "fundpulse"),
		},
		{
			filepath.Join(home, "Library", "Logs", "investments-healthcheck"),
			filepath.Join(home, "Library", "Logs", "fundpulse"),
		},
	}

	for _, d := range dirs {
		if _, err := os.Stat(d.new); err == nil {
			continue
		}
		if _, err := os.Stat(d.old); err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(d.new), 0o755); err != nil {
			continue
		}
		if err := os.Rename(d.old, d.new); err != nil {
			fmt.Fprintf(w, "warning: could not migrate %s → %s: %v\n", d.old, d.new, err)
			continue
		}
		fmt.Fprintf(w, "→ Migrated %s → %s\n", d.old, d.new)
	}

	oldPlist := filepath.Join(home, "Library", "LaunchAgents", "com.user.investments-healthcheck.plist")
	if _, err := os.Stat(oldPlist); err == nil {
		_ = exec.Command("launchctl", "bootout", "gui/"+strconv.Itoa(os.Getuid()), oldPlist).Run()
		if err := os.Remove(oldPlist); err == nil {
			fmt.Fprintln(w, "→ Removed legacy launchd plist (com.user.investments-healthcheck). Run `fundpulse config` (or `init`) to install the new schedule.")
		}
	}
}
