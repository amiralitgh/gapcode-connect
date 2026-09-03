package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentValidateSessionID_AllowsExistingUnlockedSession(t *testing.T) {
	codexHome := t.TempDir()
	sessionID := "thread-valid"
	writeValidationRollout(t, codexHome, sessionID)

	agent := &Agent{codexHome: codexHome}

	if !agent.ValidateSessionID(context.Background(), sessionID) {
		t.Fatal("ValidateSessionID() = false, want true for an existing unlocked session")
	}
}

func TestAgentValidateSessionID_RejectsMissingSession(t *testing.T) {
	agent := &Agent{codexHome: t.TempDir()}

	if agent.ValidateSessionID(context.Background(), "thread-missing") {
		t.Fatal("ValidateSessionID() = true, want false for a missing session")
	}
}

func writeValidationRollout(t *testing.T, codexHome, sessionID string) string {
	t.Helper()

	dir := filepath.Join(codexHome, "sessions", "2026", "09", "02")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create sessions directory: %v", err)
	}

	path := filepath.Join(dir, "rollout-"+sessionID+".jsonl")
	body := `{"type":"session_meta","payload":{"id":"` + sessionID + `","cwd":"/tmp/project"}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write rollout: %v", err)
	}
	return path
}
