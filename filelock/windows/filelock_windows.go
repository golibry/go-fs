// Package filelock provides thread-safe file locking functionality in non-blocking mode.
// It allows for acquiring exclusive locks on files without blocking indefinitely.
package windows

import (
	"github.com/golibry/go-fs/filelock"
	"os"
	"sync"
	"time"

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
	return fl.LockWithTimeout(0)
}

// LockWithTimeout attempts to acquire an exclusive lock on the file with a timeout
// If timeout is <= 0, it's a non-blocking operation
// If timeout is > 0, it will retry in a non-blocking manner until the timeout is reached
func (fl *FileLock) LockWithTimeout(timeout time.Duration) error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	if fl.locked {
		return filelock.ErrAlreadyLocked
	}

	var err error
	fl.file, err = os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	// Try to acquire the lock
	err = fl.tryLock(timeout)
	if err != nil {
		_ = fl.file.Close()
		fl.file = nil
		return err
	}

	fl.locked = true
	return nil
}

// tryLock attempts to acquire the lock with the specified timeout
// It uses a non-blocking approach for all cases
func (fl *FileLock) tryLock(timeout time.Duration) error {
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

	// At this point, we know the lock is held (err == windows.ERROR_LOCK_VIOLATION)

	// If timeout <= 0, it's a non-blocking call, so return immediately
	if timeout <= 0 {
		return filelock.ErrLockHeld
	}

	// For timeout > 0, retry with polling until timeout
	startTime := time.Now()
	retryInterval := time.Millisecond * 10 // Start with 10ms retry interval

	for {
		// Check if we've exceeded the timeout
		if time.Since(startTime) >= timeout {
			return filelock.ErrTimeout
		}

		// Sleep for a short interval before retrying
		time.Sleep(retryInterval)

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
