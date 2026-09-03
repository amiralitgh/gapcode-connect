//go:build unix

package codex

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func threadWriterLockHeld(codexHome, sessionID string) bool {
	held, err := probeThreadWriterLock(filepath.Join(codexHome, "thread-writer-locks", sessionID+".lock"))
	return held || err != nil
}

func probeThreadWriterLock(path string) (bool, error) {
	lockFile, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return true, nil
		}
		return false, err
	}

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		return false, err
	}
	return false, nil
}
