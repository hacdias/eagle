package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/hacdias/eagle/eagle"
	"github.com/jackc/pgx/v4"
)

func (d *Postgres) Enqueue(queue string, data []byte) error {
	return d.EnqueueAt(queue, data, time.Now().UTC())
}

func (d *Postgres) EnqueueAt(queue string, data []byte, schedule time.Time) error {
	_, err := d.pool.Exec(
		context.Background(),
		"insert into queue(queue, data, scheduled_at) values($1, $2, $3);",
		queue, data, schedule.UTC(),
	)
	return err
}

type queueItem struct {
	data      string
	attempt   int
	scheduled time.Time
}

func (d *Postgres) Dequeue(queue string, fn eagle.QueueFunc) error {
	tx, err := d.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	item, err := func() (*queueItem, error) {
		rows, err := tx.Query(context.Background(), `
delete from queue
where id = (
	select id
	from queue
	where queue = $1 and scheduled_at <= now()
	order by scheduled_at
	for update skip locked
	limit 1
)
returning data, attempt, scheduled_at;
`, queue)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		isRow := rows.Next()
		if !isRow {
			return nil, nil
		}

		var (
			data      string
			attempt   int
			scheduled time.Time
		)

		err = rows.Scan(&data, &attempt, &scheduled)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}

			return nil, err
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}

		return &queueItem{
			data:      data,
			attempt:   attempt,
			scheduled: scheduled,
		}, nil
	}()
	if err != nil {
		return err
	}
	if item == nil {
		return nil
	}

	dur := fn([]byte(item.data), item.attempt)
	if dur != 0 {
		_, err = tx.Exec(
			context.Background(),
			"insert into queue(queue, data, attempt, scheduled_at) values($1, $2, $3, $4)",
			queue, item.data, item.attempt+1, item.scheduled.Add(dur),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(context.Background())
}

func (d *Postgres) Listen(queue string, wait time.Duration, fn eagle.QueueFunc) {
	if fn == nil {
		return
	}

	d.wg.Add(1)
	for {
		select {
		case <-time.After(wait):
			err := d.Dequeue(queue, fn)
			if err != nil {
				// q.log(fmt.Errorf("could not reschedule: %w", err))
			}
		case <-d.ctx.Done():
			d.wg.Done()
			return
		}
	}
}
