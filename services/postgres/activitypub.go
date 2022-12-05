package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/hacdias/eagle/activitypub"
	"github.com/jackc/pgx/v5"
)

func (d *Postgres) AddOrUpdateFollower(follower activitypub.Follower) error {
	_, err := d.pool.Exec(
		context.Background(),
		"insert into activitypub_followers(name, id, inbox, handle) values($1, $2, $3, $4) on conflict (id) do update set name=$1, inbox=$3, handle=$4",
		follower.Name, follower.ID, follower.Inbox, follower.Handle,
	)
	return err
}

func (d *Postgres) getFollowers(sql string, args ...interface{}) ([]*activitypub.Follower, error) {
	rows, err := d.pool.Query(context.Background(), sql, args...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*activitypub.Follower{}, nil
		}

		return nil, err
	}
	defer rows.Close()

	followers := []*activitypub.Follower{}

	for rows.Next() {
		var (
			name   string
			id     string
			inbox  string
			handle string
		)
		err := rows.Scan(&name, &id, &inbox, &handle)
		if err != nil {
			return nil, err
		}

		followers = append(followers, &activitypub.Follower{
			Name:   name,
			ID:     id,
			Inbox:  inbox,
			Handle: handle,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return followers, nil
}

func (d *Postgres) GetFollower(id string) (*activitypub.Follower, error) {
	followers, err := d.getFollowers("select name, id, inbox, handle from activitypub_followers where id=$1", id)
	if err != nil {
		return nil, err
	}

	if len(followers) != 1 {
		return nil, nil
	}

	return followers[0], nil
}

func (d *Postgres) GetFollowers() ([]*activitypub.Follower, error) {
	return d.getFollowers("select name, id, inbox, handle from activitypub_followers order by id")
}

func (d *Postgres) GetFollowersByPage(page, limit int) ([]*activitypub.Follower, error) {
	return d.getFollowers("select name, id, inbox, handle from activitypub_followers order by id offset " + strconv.Itoa((page-1)*limit) + " limit " + strconv.Itoa(limit))
}

func (d *Postgres) GetFollowersCount() (int, error) {
	sql := `select count(*) from activitypub_followers;`

	rows, err := d.pool.Query(context.Background(), sql)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var n int

	_ = rows.Next()

	err = rows.Scan(&n)
	if err != nil {
		return 0, err
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	return n, nil
}

func (d *Postgres) DeleteFollower(id string) error {
	_, err := d.pool.Exec(context.Background(), "delete from activitypub_followers where id=$1", id)
	return err
}

func (d *Postgres) AddActivityPubLink(entry, activity string) error {
	_, err := d.pool.Exec(context.Background(), "insert into activitypub_links(entry_id, object_id) values($1, $2) on conflict do nothing;", entry, activity)
	return err
}

func (d *Postgres) GetActivityPubLinks(activity string) ([]string, error) {
	rows, err := d.pool.Query(context.Background(), "select entry_id from activitypub_links where object_id=$1", activity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []string{}, nil
		}

		return nil, err
	}
	defer rows.Close()

	entries := []string{}

	for rows.Next() {
		var entry string
		err := rows.Scan(&entry)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (d *Postgres) DeleteActivityPubLinks(activity string) error {
	_, err := d.pool.Exec(context.Background(), "delete from activitypub_links where object_id=$1", activity)
	return err
}
