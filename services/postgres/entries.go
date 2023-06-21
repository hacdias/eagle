package postgres

import (
	"context"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/core"
	"github.com/jackc/pgx/v5"
)

func (d *Postgres) Add(entries ...*core.Entry) error {
	b := &pgx.Batch{}

	for _, entry := range entries {
		content := entry.Title + " " + entry.Description + " " + entry.TextContent()

		updated := entry.Date.UTC()
		if !entry.LastMod.IsZero() {
			updated = entry.LastMod.UTC()
		}

		b.Queue("delete from entries where id=$1", entry.ID)
		b.Queue("insert into entries(id, content, isDraft, isDeleted, isUnlisted, published_at, updated_at) values($1, $2, $3, $4, $5, $6, $7)",
			entry.ID, content, entry.Draft, entry.Deleted(), entry.NoIndex, entry.Date.UTC(), updated)
	}

	batch := d.pool.SendBatch(context.Background(), b)
	defer batch.Close()

	for i := 0; i < b.Len(); i++ {
		_, err := batch.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Postgres) Remove(ids ...string) {
	for _, id := range ids {
		_, _ = d.pool.Exec(context.Background(), "delete from entries where id=$1", id)
	}
}

func (d *Postgres) GetSearch(opts *core.Query, query string) ([]string, error) {
	mainSelect := "select ts_rank_cd(ts, plainto_tsquery('english', $1)) as score, id, isDraft, isDeleted, isUnlisted"
	mainFrom := "from entries as e"

	sql := "select distinct (id) id, score from (" + mainSelect + " " + mainFrom + ") s where score > 0"

	args := []interface{}{query}
	where, wargs := d.whereConstraints(opts, 1)
	args = append(args, wargs...)

	if len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
	}

	sql += ` order by score desc` + d.offset(&opts.Pagination)
	return d.queryEntries(sql, 1, args...)
}

func (d *Postgres) ClearEntries() {
	_, _ = d.pool.Exec(context.Background(), "truncate table entries cascade")
}

func (d *Postgres) whereConstraints(opts *core.Query, i int) ([]string, []interface{}) {
	var where []string
	var args []interface{}

	if !opts.WithDeleted {
		where = append(where, "isDeleted=false")
	}

	if !opts.WithUnlisted {
		where = append(where, "isUnlisted=false")
	}

	if !opts.WithDrafts {
		where = append(where, "isDraft=false")
	}

	return where, args
}

func (d *Postgres) offset(opts *core.Pagination) string {
	if opts == nil {
		return ""
	}

	var sql string

	if opts.Page > 0 {
		sql += " offset " + strconv.Itoa(opts.Page*opts.Limit)
	}

	return sql + " limit " + strconv.Itoa(opts.Limit)
}

func (d *Postgres) queryEntries(sql string, ignore int, args ...interface{}) ([]string, error) {
	rows, err := d.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}

	for rows.Next() {
		var id string
		dest := make([]interface{}, ignore+1)
		dest[0] = &id
		for i := 1; i <= ignore; i++ {
			dest[1] = nil
		}
		err := rows.Scan(dest...)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}
