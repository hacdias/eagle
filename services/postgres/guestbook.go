package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/hacdias/eagle/core"
	"github.com/jackc/pgx/v5"
)

func (d *Postgres) AddGuestbookEntry(ctx context.Context, entry *core.GuestbookEntry) error {
	_, err := d.pool.Exec(
		ctx,
		"insert into guestbook_entries(name, website, content, date) values($1, $2, $3, $4)",
		entry.Name, entry.Website, entry.Content, entry.Date.UTC(),
	)

	return err
}

func (d *Postgres) GetGuestbookEntries(ctx context.Context) (core.GuestbookEntries, error) {
	return d.getGuestbookEntries(ctx, "select id, name, website, content, date from guestbook_entries order by id")
}

func (d *Postgres) GetGuestbookEntry(ctx context.Context, id int) (core.GuestbookEntry, error) {
	ee, err := d.getGuestbookEntries(ctx, "select id, name, website, content, date from guestbook_entries where id=$1", id)
	if err != nil {
		return core.GuestbookEntry{}, err
	}
	if len(ee) == 0 {
		return core.GuestbookEntry{}, errors.New("entry not found")
	}
	return ee[0], nil
}

func (d *Postgres) DeleteGuestbookEntry(ctx context.Context, id int) error {
	_, err := d.pool.Exec(ctx, "delete from guestbook_entries where id=$1", id)
	return err
}

func (d *Postgres) getGuestbookEntries(ctx context.Context, sql string, args ...interface{}) (core.GuestbookEntries, error) {
	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return core.GuestbookEntries{}, nil
		}

		return nil, err
	}
	defer rows.Close()

	entries := core.GuestbookEntries{}
	for rows.Next() {
		var (
			id      int
			name    string
			website string
			content string
			date    time.Time
		)

		err = rows.Scan(&id, &name, &website, &content, &date)
		if err != nil {
			return nil, err
		}

		entries = append(entries, core.GuestbookEntry{
			ID:      id,
			Name:    name,
			Website: website,
			Content: content,
			Date:    date,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
