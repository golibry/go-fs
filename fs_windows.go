package fs

import (
	"github.com/golibry/go-fs/filelock"
	"github.com/golibry/go-fs/filelock/windows"
)

// New creates a new FileLock for the specified file path
func New(path string) filelock.FileLock {
	return windows.New(path)
}
