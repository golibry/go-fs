// Package filelock provides thread-safe file locking functionality in non-blocking mode.
// It allows for acquiring exclusive locks on files without blocking indefinitely.
package windows

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/golibry/go-fs/filelock"

	"golang.org/x/sys/windows"
)

// FileLock represents a lock on a file
type FileLock struct {
	path   string
	file   *os.File
	locked bool
	mutex  sync.Mutex
}

// New creates a new FileLock for the specified file path
func New(path string) *FileLock {
	return &FileLock{
		path:   path,
		locked: false,
	}
}

// Lock acquires an exclusive lock on the file
// If the lock cannot be acquired immediately, it returns ErrLockHeld
func (fl *FileLock) Lock() error {
	return fl.lock(context.Background(), false)
}

// LockWithTimeout attempts to acquire an exclusive lock on the file with a timeout
// If timeout is <= 0, it's a non-blocking operation
// If timeout is > 0, it will retry in a non-blocking manner until the timeout is reached
func (fl *FileLock) LockWithTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return fl.Lock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := fl.LockContext(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		return filelock.ErrTimeout
	}

	return err
}

// LockContext attempts to acquire an exclusive lock until ctx is canceled.
func (fl *FileLock) LockContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return fl.lock(ctx, true)
}

func (fl *FileLock) lock(ctx context.Context, wait bool) error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	if fl.locked {
		return filelock.ErrAlreadyLocked
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	var err error
	fl.file, err = os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	// Try to acquire the lock
	err = fl.tryLock(ctx, wait)
	if err != nil {
		_ = fl.file.Close()
		fl.file = nil
		return err
	}

	fl.locked = true
	return nil
}

func (fl *FileLock) tryLock(ctx context.Context, wait bool) error {
	handle := windows.Handle(fl.file.Fd())
	overlapped := &windows.Overlapped{}

	// For non-blocking mode or immediate check
	err := windows.LockFileEx(
		handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		overlapped,
	)

	// If we got the lock immediately or there was an error other than lock violation, return
	if err == nil || err != windows.ERROR_LOCK_VIOLATION {
		return err
	}

	if !wait {
		return filelock.ErrLockHeld
	}

	retryInterval := time.Millisecond * 10 // Start with 10ms retry interval

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}

		// Increase retry interval for exponential backoff, but cap it at 100ms
		if retryInterval < time.Millisecond*100 {
			retryInterval = time.Duration(float64(retryInterval) * 1.5)
		}

		// Try to acquire the lock again (non-blocking)
		err = windows.LockFileEx(
			handle,
			windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
			0,
			1,
			0,
			overlapped,
		)

		// If we got the lock or there was an error other than lock violation, return
		if err == nil || err != windows.ERROR_LOCK_VIOLATION {
			return err
		}
	}
}

// Unlock releases the lock on the file
func (fl *FileLock) Unlock() error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	if !fl.locked || fl.file == nil {
		return filelock.ErrNotLocked
	}

	// Release the lock
	handle := windows.Handle(fl.file.Fd())
	overlapped := &windows.Overlapped{}
	err := windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
	if err != nil {
		return err
	}

	// Close the file
	err = fl.file.Close()
	fl.file = nil
	fl.locked = false
	return err
}

// IsLocked returns whether the file is currently locked by this process
func (fl *FileLock) IsLocked() bool {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()
	return fl.locked
}

// Path returns the file path associated with this lock
func (fl *FileLock) Path() string {
	return fl.path
}
