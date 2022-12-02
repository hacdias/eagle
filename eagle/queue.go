package eagle

import (
	"io"
	"time"
)

type QueueFunc func(content []byte, attempt int) time.Duration

type Queue interface {
	io.Closer

	Enqueue(queue string, content []byte) error
	Listen(queue string, wait time.Duration, fn QueueFunc)
}
