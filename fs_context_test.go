package fs

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golibry/go-fs/filelock"
	"github.com/stretchr/testify/require"
)

// Done is first consulted after a failed non-blocking acquisition.
type observedWaitContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (ctx *observedWaitContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.waiting) })
	return ctx.Context.Done()
}

func startObservedWait(t *testing.T, ctx context.Context, lock filelock.FileLock, wg *sync.WaitGroup) <-chan error {
	t.Helper()
	observed := &observedWaitContext{Context: ctx, waiting: make(chan struct{})}
	done := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		done <- lock.LockContext(observed)
	}()
	select {
	case <-observed.waiting:
	case err := <-done:
		t.Fatalf("LockContext returned before waiting: %v", err)
	case <-time.After(time.Second):
		t.Fatal("LockContext did not start waiting")
	}
	return done
}

func TestSameInstanceWaiterCancellation(t *testing.T) {
	for _, mode := range []string{"already-canceled", "cancel-while-waiting", "deadline", "timeout", "nonblocking"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "shared.lock")
			holder, shared := New(path), New(path)
			require.NoError(t, holder.Lock())
			ctx, cancel := context.WithCancel(context.Background())
			var wg sync.WaitGroup
			t.Cleanup(func() {
				cancel()
				wg.Wait()
				_ = shared.Unlock()
				_ = holder.Unlock()
			})
			first := startObservedWait(t, ctx, shared, &wg)

			waiterCtx, cancelWaiter := context.WithCancel(ctx)
			defer cancelWaiter()
			want := error(context.Canceled)
			switch mode {
			case "already-canceled":
				cancelWaiter()
			case "cancel-while-waiting":
				timer := time.AfterFunc(40*time.Millisecond, cancelWaiter)
				defer timer.Stop()
			case "deadline":
				var stop context.CancelFunc
				waiterCtx, stop = context.WithTimeout(ctx, 40*time.Millisecond)
				defer stop()
				want = context.DeadlineExceeded
			case "timeout":
				want = filelock.ErrTimeout
			case "nonblocking":
				want = filelock.ErrLockHeld
			}
			done := make(chan error, 1)
			wg.Add(1)
			go func() {
				defer wg.Done()
				switch mode {
				case "timeout":
					done <- shared.LockWithTimeout(40 * time.Millisecond)
				case "nonblocking":
					done <- shared.Lock()
				default:
					done <- shared.LockContext(waiterCtx)
				}
			}()
			select {
			case err := <-done:
				require.ErrorIs(t, err, want)
			case <-time.After(time.Second):
				t.Fatal("second call blocked behind the same instance's retry loop")
			}
			select {
			case err := <-first:
				t.Fatalf("first waiter unexpectedly returned: %v", err)
			default:
			}
			require.NoError(t, holder.Unlock())
			select {
			case err := <-first:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("first waiter did not acquire the released lock")
			}
			require.True(t, shared.IsLocked())
			require.ErrorIs(t, holder.Lock(), filelock.ErrLockHeld)
			require.NoError(t, shared.Unlock())
			require.NoError(t, holder.Lock())
		})
	}
}

func TestSameInstanceStateAccessWhileWaiting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	holder, shared := New(path), New(path)
	require.NoError(t, holder.Lock())
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	t.Cleanup(func() {
		cancel()
		wg.Wait()
		_ = shared.Unlock()
		_ = holder.Unlock()
	})
	first := startObservedWait(t, ctx, shared, &wg)
	done := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if shared.IsLocked() {
			done <- errors.New("waiting instance reports a held lock")
			return
		}
		done <- shared.Unlock()
	}()
	select {
	case err := <-done:
		require.ErrorIs(t, err, filelock.ErrNotLocked)
	case <-time.After(time.Second):
		t.Fatal("state access blocked behind the retry loop")
	}
	second := startObservedWait(t, ctx, shared, &wg)
	require.NoError(t, holder.Unlock())
	var results []error
	for _, result := range []<-chan error{first, second} {
		select {
		case err := <-result:
			results = append(results, err)
		case <-time.After(time.Second):
			t.Fatal("same-instance waiters did not finish")
		}
	}
	if results[0] != nil {
		results[0], results[1] = results[1], results[0]
	}
	require.NoError(t, results[0])
	require.ErrorIs(t, results[1], filelock.ErrAlreadyLocked)
	require.True(t, shared.IsLocked())
	require.ErrorIs(t, holder.Lock(), filelock.ErrLockHeld)

	cancel()
	require.ErrorIs(t, shared.LockContext(ctx), filelock.ErrAlreadyLocked)
	require.ErrorIs(t, shared.LockWithTimeout(time.Second), filelock.ErrAlreadyLocked)
	require.ErrorIs(t, shared.Lock(), filelock.ErrAlreadyLocked)
	require.NoError(t, shared.Unlock())
	require.NoError(t, holder.Lock())
}

func TestSameInstanceConcurrentUnlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unlock.lock")
	lock := New(path)
	require.NoError(t, lock.Lock())
	t.Cleanup(func() { _ = lock.Unlock() })
	const callers = 16
	start := make(chan struct{})
	results := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- lock.Unlock()
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	unlocked := 0
	for err := range results {
		if err == nil {
			unlocked++
		} else {
			require.ErrorIs(t, err, filelock.ErrNotLocked)
		}
	}
	require.Equal(t, 1, unlocked)
	require.False(t, lock.IsLocked())
	other := New(path)
	require.NoError(t, other.Lock())
	require.NoError(t, other.Unlock())
}
