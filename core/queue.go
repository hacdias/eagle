package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.hacdias.com/eagle/log"
	"go.uber.org/zap"
)

const (
	queueMaxAttempts  = 3
	queueBatchSize    = 3
	queuePollInterval = 30 * time.Second
	queueRetryDelay   = 10 * time.Minute
)

type QueueItem struct {
	ID           string
	Type         string `gorm:"index"`
	Payload      string // JSON-encoded job data
	Attempts     int
	FailedReason string
	Created      time.Time
	LastAttempt  *time.Time
}

type Queue struct {
	db       *Database
	handlers map[string]func(ctx context.Context, payload []byte) error
	notify   chan struct{}
	log      *zap.SugaredLogger
}

func newQueue(db *Database) *Queue {
	return &Queue{
		db:       db,
		handlers: map[string]func(ctx context.Context, payload []byte) error{},
		notify:   make(chan struct{}, 1),
		log:      log.S().Named("queue"),
	}
}

func (q *Queue) Register(typ string, handler func(ctx context.Context, payload []byte) error) {
	q.handlers[typ] = handler
}

func (q *Queue) Enqueue(ctx context.Context, typ string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	item := &QueueItem{
		ID:      uuid.New().String(),
		Type:    typ,
		Payload: string(data),
		Created: time.Now(),
	}

	if err := q.db.CreateQueueItem(ctx, item); err != nil {
		return err
	}

	// Non-blocking notify: if the channel already has a pending signal, skip.
	select {
	case q.notify <- struct{}{}:
	default:
	}

	return nil
}

func (q *Queue) Run(ctx context.Context) {
	ticker := time.NewTicker(queuePollInterval)
	defer ticker.Stop()

	q.processPending(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.processPending(ctx)
		case <-q.notify:
			q.processPending(ctx)
		}
	}
}

func (q *Queue) processPending(ctx context.Context) {
	retryAfter := time.Now().Add(-queueRetryDelay)
	items, err := q.db.GetPendingQueueItems(ctx, queueBatchSize, queueMaxAttempts, retryAfter)
	if err != nil {
		q.log.Errorw("failed to fetch pending items", "err", err)
		return
	}

	for _, item := range items {
		if ctx.Err() != nil {
			return
		}
		q.processItem(ctx, item)
	}
}

func (q *Queue) processItem(ctx context.Context, item *QueueItem) {
	handler, ok := q.handlers[item.Type]
	if !ok {
		q.log.Warnw("no handler registered for type", "type", item.Type, "id", item.ID)
		return
	}

	err := handler(ctx, []byte(item.Payload))
	if err == nil {
		if delErr := q.db.DeleteQueueItem(ctx, item.ID); delErr != nil {
			q.log.Errorw("failed to delete processed item", "id", item.ID, "err", delErr)
		}
		return
	}

	q.log.Errorw("handler failed", "type", item.Type, "id", item.ID, "attempt", item.Attempts+1, "err", err)
	now := time.Now()
	item.Attempts++
	item.LastAttempt = &now
	item.FailedReason = err.Error()

	if item.Attempts >= queueMaxAttempts {
		q.log.Errorw("item permanently failed", "type", item.Type, "id", item.ID)
	}

	if updateErr := q.db.UpdateQueueItem(ctx, item); updateErr != nil {
		q.log.Errorw("failed to update item after failure", "id", item.ID, "err", updateErr)
	}
}
