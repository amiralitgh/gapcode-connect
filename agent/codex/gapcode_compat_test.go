package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCodexHomeDirPrefersGapCodeHomeWhenCodexHomeIsUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("GAPCODE_HOME", "")

	gapcodeHome := filepath.Join(home, ".gapcode")
	if err := os.MkdirAll(gapcodeHome, 0o755); err != nil {
		t.Fatal(err)
	}

	if got := resolveCodexHomeDir(""); got != gapcodeHome {
		t.Fatalf("resolveCodexHomeDir() = %q, want %q", got, gapcodeHome)
	}
}

func TestResolveCodexHomeDirHonorsExplicitAndEnvironmentOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GAPCODE_HOME", filepath.Join(home, "gapcode"))

	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
	if got := resolveCodexHomeDir(""); got != filepath.Join(home, "codex") {
		t.Fatalf("CODEX_HOME result = %q", got)
	}

	explicit := filepath.Join(home, "explicit")
	if got := resolveCodexHomeDir(explicit); got != explicit {
		t.Fatalf("explicit result = %q, want %q", got, explicit)
	}
}

func TestAgentListAllSessionsIncludesOtherWorkingDirectories(t *testing.T) {
	home := t.TempDir()
	firstDir := filepath.Join(home, "first")
	secondDir := filepath.Join(home, "second")
	sessionsDir := filepath.Join(home, "sessions", "2026", "09", "02")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeSession := func(id, cwd string) {
		t.Helper()
		cwdJSON, err := json.Marshal(cwd)
		if err != nil {
			t.Fatal(err)
		}
		body := `{"type":"session_meta","payload":{"id":"` + id + `","cwd":` + string(cwdJSON) + `,"source":"cli"}}` + "\n"
		if err := os.WriteFile(filepath.Join(sessionsDir, "rollout-"+id+".jsonl"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSession("first-session", firstDir)
	writeSession("second-session", secondDir)

	agent := &Agent{workDir: firstDir, codexHome: home}
	sessions, err := agent.ListAllSessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListAllSessions() returned %d sessions, want 2", len(sessions))
	}
}

func TestSetSessionTitleAppendsGapCodeSessionIndexEntry(t *testing.T) {
	home := t.TempDir()
	agent := &Agent{codexHome: home}

	if err := agent.SetSessionTitle(context.Background(), "session-1", "Bale work"); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join(home, "session_index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var entry struct {
		ID         string `json:"id"`
		ThreadName string `json:"thread_name"`
	}
	if err := json.NewDecoder(bufio.NewReader(f)).Decode(&entry); err != nil {
		t.Fatal(err)
	}
	if entry.ID != "session-1" || entry.ThreadName != "Bale work" {
		t.Fatalf("session index entry = %+v", entry)
	}
}

func TestParseCodexReasoningEffortsJSONKeepsGapCodeCapabilities(t *testing.T) {
	data := []byte(`{"models":[{"slug":"gpt-5.6-luna","display_name":"GPT-5.6 Luna","visibility":"list","supported_in_api":true,"supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"ultra"}]}]}`)

	got := parseCodexReasoningEffortsJSON(data, "gpt-5.6-luna")
	want := []string{"low", "high", "ultra"}
	if len(got) != len(want) {
		t.Fatalf("reasoning efforts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("reasoning effort %d = %q, want %q", i, got[i], want[i])
		}
	}
}
