// Package filelock provides thread-safe file locking functionality in non-blocking mode.
// It allows for acquiring exclusive locks on files without blocking indefinitely.
package filelock

import (
	"errors"
	"time"
)

var (
	// ErrTimeout is returned when a lock operation times out
	ErrTimeout = errors.New("timeout acquiring lock")

	// ErrLockHeld is returned when a non-blocking lock operation fails because the lock is held
	ErrLockHeld = errors.New("lock is held by another process")

	// ErrAlreadyLocked is returned when trying to lock a file that
	// is already locked by this process
	ErrAlreadyLocked = errors.New("file is already locked by this process")

	// ErrNotLocked is returned when trying to unlock a file that is not locked
	ErrNotLocked = errors.New("file is not locked")
)

// FileLock defines a common interface for file locking mechanisms.
type FileLock interface {
	// Lock attempts to acquire an exclusive lock on the file.
	// Returns ErrLockHeld if the lock is already held by another process.
	Lock() error

	// LockWithTimeout attempts to acquire an exclusive lock on the file with a timeout.
	// If timeout is <= 0, it's a non-blocking operation.
	LockWithTimeout(timeout time.Duration) error

	// Unlock releases the lock on the file.
	// Returns ErrNotLocked if the file is not locked.
	Unlock() error

	// IsLocked returns true if the file is currently locked by this process.
	IsLocked() bool

	// Path returns the path to the locked file.
	Path() string
}
