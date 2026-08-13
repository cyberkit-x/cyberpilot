package platform

import (
	"fmt"
	"os"

	"github.com/gofrs/flock"
)

type Lock struct {
	file *flock.Flock
}

func AcquireLock(path string) (*Lock, error) {
	file := flock.New(path)
	locked, err := file.TryLock()
	if err != nil {
		return nil, fmt.Errorf("acquire daemon lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("another CyberPilot daemon owns %s", path)
	}
	if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
		_ = file.Unlock()
		return nil, fmt.Errorf("protect daemon lock: %w", err)
	}
	return &Lock{file: file}, nil
}

func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Unlock()
}
