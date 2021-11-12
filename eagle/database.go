package eagle

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hacdias/eagle/v2/entry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func (e *Eagle) setupPostgres() (err error) {
	dsn := "user=" + e.Config.PostgreSQL.User
	dsn += " password=" + e.Config.PostgreSQL.Password
	dsn += " host=" + e.Config.PostgreSQL.Host
	dsn += " port=" + e.Config.PostgreSQL.Port
	dsn += " dbname=" + e.Config.PostgreSQL.Database
	dsn += " pool_max_conns=10"

	e.conn, err = pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		return err
	}

	go func() {
		entries, err := e.GetEntries()
		if err != nil {
			e.Notifier.Error(err)
			return
		}

		start := time.Now()
		err = e.IndexAdd(entries...)
		if err != nil {
			e.Notifier.Error(err)
		}
		e.log.Infof("database update took %dms", time.Since(start).Milliseconds())
	}()

	return nil
}

func (e *Eagle) indexRemove(id string) {
	_, _ = e.conn.Exec(context.Background(), "delete from entries where id=$1", id)
}

func (e *Eagle) IndexAdd(entries ...*entry.Entry) error {
	b := &pgx.Batch{}

	for _, entry := range entries {
		content := entry.Title + " " + entry.Description + " " + entry.TextContent()

		b.Queue("delete from entries where id=$1", entry.ID)
		b.Queue("insert into entries(id, content, isDraft, isDeleted, isPrivate, date) values($1, $2, $3, $4, $5, $6)", entry.ID, content, entry.Draft, entry.Deleted, entry.Private, entry.Published.UTC())

		for _, tag := range entry.Tags() {
			b.Queue("insert into tags(entry_id, tag) values ($1, $2)", entry.ID, tag)
		}

		if len(entry.Sections) > 0 {
			for _, section := range entry.Sections {
				b.Queue("insert into sections(entry_id, section) values ($1, $2)", entry.ID, section)
			}
		}
	}

	batch := e.conn.SendBatch(context.Background(), b)
	defer batch.Close()

	for i := 0; i < b.Len(); i++ {
		_, err := batch.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

type QueryOptions struct {
	Page    int
	Draft   bool
	Deleted bool
	Private bool
}

func (e *Eagle) Tags() ([]string, error) {
	rows, err := e.conn.Query(context.Background(), "select distinct tag from tags order by tag")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []string{}

	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		tags = append(tags, id)
	}

	return tags, rows.Err()
}

func (e *Eagle) QueryDate(year, month, day int, opts *QueryOptions) ([]*entry.Entry, error) {
	i := 0
	args := []interface{}{}

	sql := "select id from entries where "
	where := []string{}

	if year > 0 {
		i++
		where = append(where, "date_part('year', date)=$"+strconv.Itoa(i))
		args = append(args, year)
	}

	if month > 0 {
		i++
		where = append(where, "date_part('month', date)=$"+strconv.Itoa(i))
		args = append(args, month)
	}

	if day > 0 {
		i++
		where = append(where, "date_part('day', date)=$"+strconv.Itoa(i))
		args = append(args, day)
	}

	sql += strings.Join(where, " and ")
	sql += e.finishQuery(opts)
	return e.queryEntries(sql, args...)
}

func (e *Eagle) QueryEntries(opts *QueryOptions) ([]*entry.Entry, error) {
	sql := "select id from entries" + e.finishQuery(opts)
	return e.queryEntries(sql)
}

func (e *Eagle) QueryTag(tag string, opts *QueryOptions) ([]*entry.Entry, error) {
	sql := "select id from tags inner join entries on id=entry_id where tag=$1" + e.finishQuery(opts)
	return e.queryEntries(sql, tag)
}

func (e *Eagle) QuerySection(sections []string, opts *QueryOptions) ([]*entry.Entry, error) {
	args := []interface{}{}
	sql := "select id from sections inner join entries on id=entry_id where "
	where := []string{}

	for i, section := range sections {
		where = append(where, "section=$"+strconv.Itoa(i+1))
		args = append(args, section)
	}

	sql += strings.Join(where, " or ")
	sql += e.finishQuery(opts)
	return e.queryEntries(sql, args...)
}

func (e *Eagle) SearchPostgres(query string, opts *QueryOptions) ([]*entry.Entry, error) {
	sql := `select id from (
		select ts_rank_cd(ts, plainto_tsquery('english', $1)) as score, e.id
		from entries as e
	) s
	where score > 0
	order by score desc`

	if opts.Page > 0 {
		sql += " offset "
		sql += strconv.Itoa(opts.Page * e.Config.Site.Paginate)
	}

	sql += " limit "
	sql += strconv.Itoa(e.Config.Site.Paginate)

	return e.queryEntries(sql, query)
}

func (e *Eagle) finishQuery(opts *QueryOptions) string {
	var query strings.Builder

	if !opts.Deleted {
		query.WriteString(" and isDeleted=false")
	}

	if !opts.Private {
		query.WriteString(" and isPrivate=false")
	}

	if !opts.Draft {
		query.WriteString(" and isDraft=false")
	}

	query.WriteString(" order by date desc")

	if opts.Page > 0 {
		query.WriteString(" offset ")
		query.WriteString(strconv.Itoa(opts.Page * e.Config.Site.Paginate))
	}

	query.WriteString(" limit ")
	query.WriteString(strconv.Itoa(e.Config.Site.Paginate))
	return query.String()
}

func (e *Eagle) queryEntries(sql string, args ...interface{}) ([]*entry.Entry, error) {
	rows, err := e.conn.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}

	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	entries := []*entry.Entry{}

	for _, id := range ids {
		entry, err := e.GetEntry(id)
		if err != nil {
			if os.IsNotExist(err) {
				e.indexRemove(id)
			} else {
				return nil, err
			}
		} else {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
