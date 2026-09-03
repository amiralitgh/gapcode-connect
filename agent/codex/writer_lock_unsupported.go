//go:build !unix

package codex

func threadWriterLockHeld(_, _ string) bool {
	return false
}
