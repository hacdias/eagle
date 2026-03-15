package core

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestQueue(t *testing.T) (*Queue, *Database) {
	t.Helper()
	f, err := os.CreateTemp("", "eagle-queue-*.db")
	require.NoError(t, err)
	_ = f.Close()
	t.Cleanup(func() {
		_ = os.Remove(f.Name())
	})

	db, err := newDatabase(f.Name())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	q := newQueue(db)
	return q, db
}

// getPendingItems returns all pending items, including recently-failed ones, by
// using a far-future retryAfter so LastAttempt filtering is bypassed.
func getPendingItems(t *testing.T, db *Database) []*QueueItem {
	t.Helper()
	items, err := db.GetPendingQueueItems(context.Background(), 100, queueMaxAttempts, time.Now().Add(time.Hour))
	require.NoError(t, err)
	return items
}

func TestQueueEnqueue(t *testing.T) {
	q, db := newTestQueue(t)
	ctx := context.Background()

	err := q.Enqueue(ctx, "test", "payload")
	require.NoError(t, err)

	// Notify channel should have received a signal.
	select {
	case <-q.notify:
		// ok
	default:
		t.Fatal("expected notify signal after enqueue")
	}

	// Item should be in DB.
	items, err := db.GetPendingQueueItems(ctx, 10, queueMaxAttempts, time.Now())
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "test", items[0].Type)
	assert.Equal(t, `"payload"`, items[0].Payload)
}

func TestQueueEnqueue_NoDoubleNotify(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	require.NoError(t, q.Enqueue(ctx, "test", 1))
	require.NoError(t, q.Enqueue(ctx, "test", 2))

	// Channel is buffered with size 1; at most one signal should be queued.
	count := 0
	for {
		select {
		case <-q.notify:
			count++
			continue
		default:
		}
		break
	}
	assert.Equal(t, 1, count)
}

func TestQueueProcessItem_Success(t *testing.T) {
	q, db := newTestQueue(t)
	ctx := context.Background()

	var got []byte
	q.Register("test", func(_ context.Context, payload []byte) error {
		got = payload
		return nil
	})

	require.NoError(t, q.Enqueue(ctx, "test", "hello"))

	items, err := db.GetPendingQueueItems(ctx, 10, queueMaxAttempts, time.Now())
	require.NoError(t, err)
	require.Len(t, items, 1)

	q.processItem(ctx, items[0])

	assert.Equal(t, `"hello"`, string(got))

	// Item must be deleted on success.
	remaining, err := db.GetPendingQueueItems(ctx, 10, queueMaxAttempts, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestQueueProcessItem_HandlerFailure(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	q.Register("test", func(_ context.Context, _ []byte) error {
		return errors.New("boom")
	})

	require.NoError(t, q.Enqueue(ctx, "test", "hello"))

	items := getPendingItems(t, q.db)
	require.Len(t, items, 1)

	q.processItem(ctx, items[0])

	// Item should still exist with attempts incremented and LastAttempt set.
	updated := getPendingItems(t, q.db)
	require.Len(t, updated, 1)
	assert.Equal(t, 1, updated[0].Attempts)
	assert.NotNil(t, updated[0].LastAttempt)
}

func TestQueueProcessItem_RespectsRetryDelay(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	callCount := 0
	q.Register("test", func(_ context.Context, _ []byte) error {
		callCount++
		return errors.New("boom")
	})

	require.NoError(t, q.Enqueue(ctx, "test", "hello"))

	// First attempt.
	items := getPendingItems(t, q.db)
	require.Len(t, items, 1)
	q.processItem(ctx, items[0])
	assert.Equal(t, 1, callCount)

	// processPending with the real retryAfter should not pick up the item again
	// because its LastAttempt is too recent.
	q.processPending(ctx)
	assert.Equal(t, 1, callCount, "item should not be retried before retry delay elapses")
}

func TestQueueProcessItem_MaxAttempts(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	q.Register("test", func(_ context.Context, _ []byte) error {
		return errors.New("always fails")
	})

	require.NoError(t, q.Enqueue(ctx, "test", "hello"))

	for i := 0; i < queueMaxAttempts; i++ {
		items := getPendingItems(t, q.db)
		require.Len(t, items, 1, "item should still be pending on attempt %d", i+1)
		q.processItem(ctx, items[0])
	}

	// After max attempts the item should no longer appear in pending.
	pending := getPendingItems(t, q.db)
	assert.Empty(t, pending)

	// But it should be retrievable via GetFailedQueueItems.
	failed, err := q.db.GetFailedQueueItems(ctx)
	require.NoError(t, err)
	require.Len(t, failed, 1)
	assert.Equal(t, queueMaxAttempts, failed[0].Attempts)
}

func TestQueueProcessItem_NoHandler(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	// No handler registered for "unknown".
	require.NoError(t, q.Enqueue(ctx, "unknown", "hello"))

	items := getPendingItems(t, q.db)
	require.Len(t, items, 1)

	q.processItem(ctx, items[0])

	// Item should remain unchanged (no attempts incremented, no delete).
	remaining := getPendingItems(t, q.db)
	require.Len(t, remaining, 1)
	assert.Equal(t, 0, remaining[0].Attempts)
	assert.Nil(t, remaining[0].LastAttempt)
}

func TestQueueProcessPending_BatchSize(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx := context.Background()

	callCount := 0
	q.Register("test", func(_ context.Context, _ []byte) error {
		callCount++
		return nil
	})

	// Enqueue more items than the batch size.
	for i := 0; i < queueBatchSize+2; i++ {
		require.NoError(t, q.Enqueue(ctx, "test", i))
	}

	q.processPending(ctx)

	assert.Equal(t, queueBatchSize, callCount, "should process exactly one batch")

	// Remaining items are still in the DB.
	remaining := getPendingItems(t, q.db)
	assert.Len(t, remaining, 2)
}

func TestQueueProcessPending_StopsOnContextCancel(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	q.Register("test", func(_ context.Context, _ []byte) error {
		callCount++
		cancel() // cancel after first item
		return nil
	})

	for i := 0; i < queueBatchSize; i++ {
		require.NoError(t, q.Enqueue(context.Background(), "test", i))
	}

	q.processPending(ctx)

	assert.Equal(t, 1, callCount, "should stop processing after context is cancelled")
}

func TestQueueRun_StopsOnContextCancel(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		q.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
}

func TestQueueRun_ProcessesOnNotify(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processed := make(chan struct{})
	q.Register("test", func(_ context.Context, _ []byte) error {
		select {
		case processed <- struct{}{}:
		default:
		}
		return nil
	})

	go q.Run(ctx)

	// Enqueue triggers a notify signal; Run should pick it up without waiting for the ticker.
	require.NoError(t, q.Enqueue(ctx, "test", "hello"))

	select {
	case <-processed:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("item was not processed promptly after enqueue notification")
	}
}

func TestQueueRun_ProcessesExistingItemsOnStart(t *testing.T) {
	q, _ := newTestQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Enqueue before Run starts (no notify will be sent on startup).
	require.NoError(t, q.Enqueue(context.Background(), "test", "hello"))

	processed := make(chan struct{})
	q.Register("test", func(_ context.Context, _ []byte) error {
		select {
		case processed <- struct{}{}:
		default:
		}
		return nil
	})

	go q.Run(ctx)

	select {
	case <-processed:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("pre-existing item was not processed on Run startup")
	}
}
