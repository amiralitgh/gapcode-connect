package codex

import (
	"context"
	"fmt"
	"strings"
)

// ValidateSessionID reports whether sessionID belongs to this Codex state
// directory and is not currently owned by another writer.
func (a *Agent) ValidateSessionID(_ context.Context, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}

	a.mu.RLock()
	codexHome := a.codexHome
	a.mu.RUnlock()

	if findSessionFile(sessionID, codexHome) == "" {
		return false
	}

	return !threadWriterLockHeld(resolveCodexHomeDir(codexHome), sessionID)
}

func sessionWriterLockError(sessionID string) error {
	return fmt.Errorf("codex: session %q is currently owned by another writer", sessionID)
}
