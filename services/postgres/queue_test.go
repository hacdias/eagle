package postgres

import (
	"testing"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	q, err := NewPostgres(&eagle.PostgreSQL{
		User:     "postgres",
		Password: "pgpassword",
		Host:     "127.0.0.1",
		Port:     "5432",
		Database: "eagle",
	})
	if err != nil {
		t.Error(err)
	}

	time1 := time.Now()

	err = q.EnqueueAt("test", []byte("1"), time1)
	require.NoError(t, err)

	err = q.EnqueueAt("test", []byte("2"), time.Now())
	require.NoError(t, err)

	err = q.Dequeue("abc", func(content []byte, attempt int) time.Duration {
		t.Fail()
		return 0
	})
	require.NoError(t, err)

	err = q.Dequeue("test", func(content []byte, attempt int) time.Duration {
		require.Equal(t, []byte("1"), content)
		require.Equal(t, 1, attempt)
		return time.Second
	})
	require.NoError(t, err)

	err = q.Dequeue("test", func(content []byte, attempt int) time.Duration {
		require.Equal(t, []byte("2"), content)
		require.Equal(t, 1, attempt)
		return 0
	})
	require.NoError(t, err)

	err = q.Dequeue("test", func(content []byte, attempt int) time.Duration {
		t.Fail()
		return 0
	})
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	err = q.Dequeue("test", func(content []byte, attempt int) time.Duration {
		require.Equal(t, []byte("1"), content)
		require.Equal(t, 2, attempt)
		return 0
	})
	require.NoError(t, err)
}

// func TestQueueListen(t *testing.T) {
// 	dir, err := os.MkdirTemp("", "eagle")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer os.RemoveAll(dir)

// 	q, err := NewQueue(dir, nil)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer q.Close()

// 	countA := 0
// 	countB := 0

// 	go q.Listen("a", time.Second, func(item *Item, dequeue func(), reschedule func(time.Duration)) {
// 		countA++
// 		dequeue()
// 	})

// 	go q.Listen("b", time.Second, func(item *Item, dequeue func(), reschedule func(time.Duration)) {
// 		countB++
// 		dequeue()
// 	})

// 	now := time.Now()

// 	q.Enqueue("a", []byte("A"), now)
// 	q.Enqueue("a", []byte("B"), now.Add(time.Second*-1))

// 	q.Enqueue("b", []byte("A"), now)
// 	q.Enqueue("b", []byte("B"), now.Add(time.Second*2))
// 	q.Enqueue("b", []byte("C"), now.Add(time.Second*10))

// 	time.Sleep(time.Second * 3)

// 	require.Equal(t, 2, countA)
// 	require.Equal(t, 2, countB)
// }
