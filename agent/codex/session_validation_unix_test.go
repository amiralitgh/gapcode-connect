//go:build unix

package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestAgentValidateSessionID_RejectsExternallyHeldWriterLock(t *testing.T) {
	codexHome := t.TempDir()
	sessionID := "thread-externally-active"
	writeValidationRollout(t, codexHome, sessionID)

	lockDir := filepath.Join(codexHome, "thread-writer-locks")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("create writer lock directory: %v", err)
	}
	lockPath := filepath.Join(lockDir, sessionID+".lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open writer lock: %v", err)
	}
	defer lockFile.Close()
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("hold writer lock: %v", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	agent := &Agent{codexHome: codexHome}

	if agent.ValidateSessionID(context.Background(), sessionID) {
		t.Fatal("ValidateSessionID() = true, want false while an external writer lock is held")
	}
}

func TestAgentStartSession_RejectsExternallyHeldWriterLock(t *testing.T) {
	codexHome := t.TempDir()
	sessionID := "thread-start-blocked"
	writeValidationRollout(t, codexHome, sessionID)

	lockDir := filepath.Join(codexHome, "thread-writer-locks")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("create writer lock directory: %v", err)
	}
	lockPath := filepath.Join(lockDir, sessionID+".lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open writer lock: %v", err)
	}
	defer lockFile.Close()
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("hold writer lock: %v", err)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	agent := &Agent{codexHome: codexHome}

	session, err := agent.StartSession(context.Background(), sessionID)
	if session != nil {
		t.Fatal("StartSession() returned a session while an external writer lock was held")
	}
	if err == nil {
		t.Fatal("StartSession() error = nil, want writer-lock error")
	}
}

func TestAgentValidateSessionID_AllowsExistingSessionWithUnlockedWriterLock(t *testing.T) {
	codexHome := t.TempDir()
	sessionID := "thread-inactive"
	writeValidationRollout(t, codexHome, sessionID)

	lockDir := filepath.Join(codexHome, "thread-writer-locks")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("create writer lock directory: %v", err)
	}
	lockPath := filepath.Join(lockDir, sessionID+".lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open writer lock: %v", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("hold and release writer lock: %v", err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatalf("release writer lock: %v", err)
	}

	agent := &Agent{codexHome: codexHome}

	if !agent.ValidateSessionID(context.Background(), sessionID) {
		t.Fatal("ValidateSessionID() = false, want true after the writer lock is released")
	}
}

func TestProbeThreadWriterLock_IgnoresMissingLock(t *testing.T) {
	held, err := probeThreadWriterLock(filepath.Join(t.TempDir(), "missing.lock"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("probeThreadWriterLock() error = %v, want nil or os.ErrNotExist", err)
	}
	if held {
		t.Fatal("probeThreadWriterLock() = true, want false for a missing lock")
	}
}
