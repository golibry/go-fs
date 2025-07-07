# go-fs

Golang file system common functionalities.  
Migrated from https://github.com/rsgcata/go-fs

## Installation

```bash
go get github.com/golibry/go-fs
```

## Packages

### fs

The `fs` package provides a platform-agnostic way to create file locks.

#### Usage

```go
import (
	"github.com/golibry/go-fs"
)

// Create a new file lock (automatically uses the appropriate implementation for your OS)
lock := fs.New("myfile.lock")
```

#### API Reference

**New Function**

```go
// New creates a new FileLock for the specified file path
func New(path string) filelock.FileLock
```

This function returns a platform-specific implementation of the FileLock interface based on the current operating system:
- On Windows, it returns a windows.FileLock
- On Unix/Linux/macOS, it returns a unix.FileLock

### filelock

The `filelock` package provides thread-safe file locking functionality in non-blocking mode. It allows for acquiring exclusive locks on files without blocking indefinitely.

#### Features

- Thread-safe file locking
- Non-blocking and timeout-based lock acquisition
- Platform-specific implementations for Windows and Unix systems

#### Usage Examples

**Platform-Agnostic Usage (Recommended)**

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golibry/go-fs"
)

func main() {
	// Create a new file lock (automatically uses the appropriate implementation for your OS)
	lock := fs.New("myfile.lock")

	// Try to acquire the lock (non-blocking)
	err := lock.Lock()
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}

	fmt.Println("Lock acquired")

	// Do some work with the locked file
	// ...

	// Release the lock
	err = lock.Unlock()
	if err != nil {
		log.Fatalf("Failed to release lock: %v", err)
	}

	fmt.Println("Lock released")
}
```

**Platform-Specific Usage (Legacy)**

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golibry/go-fs/filelock/unix"
	// For Windows: "github.com/golibry/go-fs/filelock/windows"
)

func main() {
	// Create a new file lock
	lock := unix.New("myfile.lock")
	// For Windows: lock := windows.New("myfile.lock")

	// Try to acquire the lock (non-blocking)
	err := lock.Lock()
	if err != nil {
		log.Fatalf("Failed to acquire lock: %v", err)
	}

	fmt.Println("Lock acquired")

	// Do some work with the locked file
	// ...

	// Release the lock
	err = lock.Unlock()
	if err != nil {
		log.Fatalf("Failed to release lock: %v", err)
	}

	fmt.Println("Lock released")
}
```

**Lock with Timeout**

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golibry/go-fs"
)

func main() {
	// Create a new file lock (automatically uses the appropriate implementation for your OS)
	lock := fs.New("myfile.lock")

	// Try to acquire the lock with a 5-second timeout
	err := lock.LockWithTimeout(5 * time.Second)
	if err != nil {
		log.Fatalf("Failed to acquire lock within timeout: %v", err)
	}

	fmt.Println("Lock acquired")

	// Check if the file is locked
	if lock.IsLocked() {
		fmt.Println("File is locked by this process")
	}

	// Get the path of the locked file
	fmt.Printf("Locked file path: %s\n", lock.Path())

	// Release the lock
	err = lock.Unlock()
	if err != nil {
		log.Fatalf("Failed to release lock: %v", err)
	}

	fmt.Println("Lock released")
}
```

**Error Handling**

```go
package main

import (
	"fmt"
	"time"

	"github.com/golibry/go-fs"
	"github.com/golibry/go-fs/filelock"
)

func main() {
	// Create two lock instances for the same file
	lock1 := fs.New("myfile.lock")
	lock2 := fs.New("myfile.lock")

	// Acquire the lock with the first instance
	err := lock1.Lock()
	if err != nil {
		fmt.Printf("Failed to acquire lock1: %v\n", err)
		return
	}
	fmt.Println("Lock1 acquired")

	// Try to acquire the same lock with the second instance (should fail)
	err = lock2.Lock()
	if err != nil {
		switch err {
		case filelock.ErrLockHeld:
			fmt.Println("Lock is held by another process (expected)")
		case filelock.ErrAlreadyLocked:
			fmt.Println("File is already locked by this process")
		default:
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	// Try with timeout
	err = lock2.LockWithTimeout(2 * time.Second)
	if err == filelock.ErrTimeout {
		fmt.Println("Timeout acquiring lock (expected)")
	}

	// Release the first lock
	err = lock1.Unlock()
	if err != nil {
		fmt.Printf("Failed to release lock1: %v\n", err)
		return
	}
	fmt.Println("Lock1 released")

	// Now the second lock should succeed
	err = lock2.Lock()
	if err != nil {
		fmt.Printf("Failed to acquire lock2: %v\n", err)
		return
	}
	fmt.Println("Lock2 acquired")

	// Release the second lock
	err = lock2.Unlock()
	if err != nil {
		fmt.Printf("Failed to release lock2: %v\n", err)
		return
	}
	fmt.Println("Lock2 released")
}
```

#### API Reference

**FileLock Interface**

```go
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
```

**Error Types**

- `ErrTimeout`: Returned when a lock operation times out
- `ErrLockHeld`: Returned when a non-blocking lock operation fails because the lock is held
- `ErrAlreadyLocked`: Returned when trying to lock a file that is already locked by this process
- `ErrNotLocked`: Returned when trying to unlock a file that is not locked

**Platform-Specific Implementations**

- Unix: `github.com/golibry/go-fs/filelock/unix`
- Windows: `github.com/golibry/go-fs/filelock/windows`

Each implementation provides a `New(path string)` function that returns a new FileLock instance for the specified file path.
  
  
**See _examples folder for some basic usage**
