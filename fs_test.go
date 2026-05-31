package fs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewReturnsUsableFileLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "root.lock")
	lock := New(lockPath)

	require.Equal(t, lockPath, lock.Path())
	require.NoError(t, lock.Lock())
	require.True(t, lock.IsLocked())
	require.NoError(t, lock.Unlock())
	require.False(t, lock.IsLocked())
}
