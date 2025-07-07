package windows

import (
	"github.com/golibry/go-fs/filelock"
	"github.com/golibry/go-fs/filelock/unix"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FileLockTestSuite defines a test suite for the FileLock functionality
type FileLockTestSuite struct {
	suite.Suite
	tempDir string
}

// SetupTest creates a temporary directory for test files before each test
func (s *FileLockTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "filelock-test")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
}

// TearDownTest removes the temporary directory after each test
func (s *FileLockTestSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

// TestBasicLockAndUnlock tests the basic lock and unlock functionality
func (s *FileLockTestSuite) TestBasicLockAndUnlock() {
	lockPath := filepath.Join(s.tempDir, "basic.lock")
	lock := unix.New(lockPath)

	// Lock the file
	err := lock.Lock()
	s.Require().NoError(err)
	s.Assert().True(lock.IsLocked())

	// Unlock the file
	err = lock.Unlock()
	s.Require().NoError(err)
	s.Assert().False(lock.IsLocked())
}

// TestDoubleLock tests that locking an already locked file returns an error
func (s *FileLockTestSuite) TestDoubleLock() {
	lockPath := filepath.Join(s.tempDir, "double.lock")
	lock := unix.New(lockPath)

	// Lock the file
	err := lock.Lock()
	s.Require().NoError(err)
	s.Assert().True(lock.IsLocked())

	// Try to lock it again (should fail)
	err = lock.Lock()
	s.Assert().Equal(filelock.ErrAlreadyLocked, err)

	// Unlock the file
	err = lock.Unlock()
	s.Require().NoError(err)
	s.Assert().False(lock.IsLocked())
}

// TestUnlockWithoutLock tests that unlocking a file that isn't locked returns an error
func (s *FileLockTestSuite) TestUnlockWithoutLock() {
	lockPath := filepath.Join(s.tempDir, "unlock.lock")
	lock := unix.New(lockPath)

	// Try to unlock without locking first
	err := lock.Unlock()
	s.Assert().Equal(filelock.ErrNotLocked, err)
}

// TestConcurrentLocks tests that concurrent locks work as expected
func (s *FileLockTestSuite) TestConcurrentLocks() {
	lockPath := filepath.Join(s.tempDir, "concurrent.lock")

	// Create a lock and acquire it
	lock1 := unix.New(lockPath)
	err := lock1.Lock()
	s.Require().NoError(err)

	// Try to acquire the same lock from another instance (should fail with ErrLockHeld)
	lock2 := unix.New(lockPath)
	err = lock2.Lock()
	s.Assert().Equal(filelock.ErrLockHeld, err)

	// Release the first lock
	err = lock1.Unlock()
	s.Require().NoError(err)

	// Now the second lock should succeed
	err = lock2.Lock()
	s.Require().NoError(err)

	// Clean up
	err = lock2.Unlock()
	s.Require().NoError(err)
}

// TestLockWithTimeout tests the timeout functionality when acquiring a lock
func (s *FileLockTestSuite) TestLockWithTimeout() {
	lockPath := filepath.Join(s.tempDir, "timeout.lock")

	// Create a lock and acquire it
	lock1 := unix.New(lockPath)
	err := lock1.Lock()
	s.Require().NoError(err)

	// Try to acquire with a short timeout (should fail with ErrTimeout)
	lock2 := unix.New(lockPath)
	err = lock2.LockWithTimeout(100 * time.Millisecond)
	s.Assert().Equal(filelock.ErrTimeout, err)

	// Release the first lock
	err = lock1.Unlock()
	s.Require().NoError(err)
}

// TestNonBlockingBehavior tests that LockWithTimeout doesn't block threads indefinitely
func (s *FileLockTestSuite) TestNonBlockingBehavior() {
	lockPath := filepath.Join(s.tempDir, "nonblocking.lock")

	// Create a lock and acquire it
	lock1 := unix.New(lockPath)
	err := lock1.Lock()
	s.Require().NoError(err)
	defer lock1.Unlock()

	// Create a channel to signal when the goroutine has completed
	done := make(chan struct{})

	// Start a goroutine that tries to acquire the lock with a long timeout
	go func() {
		lock2 := unix.New(lockPath)
		// Use a relatively long timeout
		err := lock2.LockWithTimeout(500 * time.Millisecond)
		// We expect a timeout error
		if err != filelock.ErrTimeout {
			s.T().Errorf("Expected ErrTimeout, got: %v", err)
		}
		// Signal that the goroutine has completed
		close(done)
	}()

	// Wait for the goroutine to complete with a timeout that's longer than the lock timeout
	// If the goroutine is blocked indefinitely, this will timeout
	select {
	case <-done:
		// Success - the goroutine completed as expected
	case <-time.After(1 * time.Second):
		s.T().Error("LockWithTimeout appears to be blocking indefinitely")
	}
}

// TestThreadSafety tests that the FileLock is thread-safe
func (s *FileLockTestSuite) TestThreadSafety() {
	lockPath := filepath.Join(s.tempDir, "threadsafe.lock")
	lock := unix.New(lockPath)

	// Create multiple goroutines that try to lock and unlock
	var wg sync.WaitGroup
	const numGoroutines = 10

	// Channel to collect errors
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Try to lock
			err := lock.Lock()
			if err != nil && err != filelock.ErrAlreadyLocked && err != filelock.ErrLockHeld {
				errChan <- err
				return
			}

			// If we got the lock, unlock it
			if err == nil {
				time.Sleep(10 * time.Millisecond) // Simulate some work
				err = lock.Unlock()
				if err != nil {
					errChan <- err
				}
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Check if there were any errors
	for err := range errChan {
		s.T().Errorf("Unexpected error: %v", err)
	}

	// Make sure the lock is unlocked at the end
	s.Assert().False(lock.IsLocked())
}

// TestFileLock runs the test suite
func TestFileLock(t *testing.T) {
	suite.Run(t, new(FileLockTestSuite))
}
