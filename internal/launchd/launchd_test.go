package launchd

import (
	"strings"
	"testing"
)

func TestRenderPlistContainsTimingAndPaths(t *testing.T) {
	plist, err := RenderPlist(Config{
		Label:     "com.lizarusi.fundpulse",
		Binary:    "/usr/local/bin/fundpulse",
		Args:      []string{"run"},
		Hour:      18,
		Minute:    30,
		LogPath:   "/Users/u/Library/Logs/fundpulse/run.log",
		RunAtLoad: true,
	})
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	mustContain := []string{
		"<string>com.lizarusi.fundpulse</string>",
		"<string>/usr/local/bin/fundpulse</string>",
		"<string>run</string>",
		"<integer>18</integer>",
		"<integer>30</integer>",
		"/Users/u/Library/Logs/fundpulse/run.log",
		"<key>RunAtLoad</key>",
		"<key>StartCalendarInterval</key>",
	}
	for _, s := range mustContain {
		if !strings.Contains(plist, s) {
			t.Errorf("plist missing %q\n--\n%s", s, plist)
		}
	}
}

func TestParseScheduleTimeValid(t *testing.T) {
	h, m, err := ParseScheduleTime("18:30")
	if err != nil {
		t.Fatal(err)
	}
	if h != 18 || m != 30 {
		t.Errorf("got %d:%d, want 18:30", h, m)
	}
}

func TestParseScheduleTimeInvalid(t *testing.T) {
	for _, s := range []string{"", "18", "25:00", "12:60", "abc"} {
		if _, _, err := ParseScheduleTime(s); err == nil {
			t.Errorf("ParseScheduleTime(%q) should error", s)
		}
	}
}
