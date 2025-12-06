# go-fs

Golang file system common functionalities.  
Migrated from https://github.com/rsgcata/go-fs

## Features

- Cross‑platform file locking (Windows and Unix-like systems)
- Thread‑safe, non‑blocking acquisition with optional timeouts
- Simple, unified API with platform‑specific implementations under the hood

## Installation

```bash
go get github.com/golibry/go-fs
```

## Packages

### fs

The `fs` package provides a platform‑agnostic way to create file locks.

See the `_examples` folder for usage.

#### API Reference

New

- `New(path string) filelock.FileLock` — creates a new file lock for the specified path.

This function returns a platform‑specific implementation of the `filelock.FileLock` interface based on the current operating system:
- On Windows, it returns a `windows.FileLock`
- On Unix/Linux/macOS, it returns a `unix.FileLock`

### filelock

The `filelock` package provides thread‑safe file locking in non‑blocking mode. It allows acquiring exclusive locks on files without blocking indefinitely and supports timeouts.

See the `_examples` folder for usage.

#### API Reference

FileLock interface

- `Lock() error`
- `LockWithTimeout(timeout time.Duration) error`
- `Unlock() error`
- `IsLocked() bool`
- `Path() string`

Errors

- `ErrTimeout`: lock operation timed out
- `ErrLockHeld`: non‑blocking lock failed because another process holds the lock
- `ErrAlreadyLocked`: trying to lock a file already locked by this process
- `ErrNotLocked`: trying to unlock a file that is not locked

Platform‑specific implementations

- Unix: `github.com/golibry/go-fs/filelock/unix`
- Windows: `github.com/golibry/go-fs/filelock/windows`

Each implementation provides `New(path string)` that returns a new `FileLock` instance.

## Examples

See the [`_examples` folder](./_examples) for runnable snippets demonstrating common usage patterns.
