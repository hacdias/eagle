package database

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/v2/config"
	"github.com/hacdias/eagle/v2/entry"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewDatabase(cfg *config.PostgreSQL) (Database, error) {
	dsn := "user=" + cfg.User
	dsn += " password=" + cfg.Password
	dsn += " host=" + cfg.Host
	dsn += " port=" + cfg.Port
	dsn += " dbname=" + cfg.Database

	err := migrate(dsn)
	if err != nil {
		return nil, err
	}

	dsn += " pool_max_conns=10"

	pool, err := pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	return &Postgres{pool}, nil
}

func (d *Postgres) Remove(id string) {
	_, _ = d.pool.Exec(context.Background(), "delete from entries where id=$1", id)
}

func (d *Postgres) Add(entries ...*entry.Entry) error {
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

func (d *Postgres) GetTags() ([]string, error) {
	rows, err := d.pool.Query(context.Background(), "select distinct tag from tags order by tag")
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

func (d *Postgres) ByDate(opts *QueryOptions, year, month, day int) ([]string, error) {
	if year == 0 && month == 0 && day == 0 {
		return nil, errors.New("year, month or day must be set")
	}

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

	where = append(where, d.whereConstraints(opts)...)
	sql += strings.Join(where, " and ")
	sql += " order by date desc"
	sql += d.offset(opts)

	return d.queryEntries(sql, args...)
}

func (d *Postgres) ByTag(opts *QueryOptions, tag string) ([]string, error) {
	args := []interface{}{tag}
	sql := "select id from tags inner join entries on id=entry_id where tag=$1"

	if ands := d.whereConstraints(opts); len(ands) > 0 {
		sql += " and " + strings.Join(ands, " and ")
	}

	sql += " order by date desc" + d.offset(opts)
	return d.queryEntries(sql, args...)
}

func (d *Postgres) BySection(opts *QueryOptions, sections ...string) ([]string, error) {
	args := []interface{}{}
	sql := "select id from sections inner join entries on id=entry_id"
	ands := d.whereConstraints(opts)

	var sectionsWhere []string

	for i, section := range sections {
		sectionsWhere = append(sectionsWhere, "section=$"+strconv.Itoa(i+1))
		args = append(args, section)
	}

	if len(sectionsWhere) > 0 {
		ands = append(ands, "("+strings.Join(sectionsWhere, " or ")+")")
	}

	if len(ands) > 0 {
		sql += " where " + strings.Join(ands, " and ")
	}

	sql += " order by date desc" + d.offset(opts)
	return d.queryEntries(sql, args...)
}

func (d *Postgres) Search(opts *QueryOptions, query string) ([]string, error) {
	sql := `select id from (
		select ts_rank_cd(ts, plainto_tsquery('english', $1)) as score, id, isDraft, isDeleted, isPrivate
		from entries as e
	) s
	where score > 0`

	if ands := d.whereConstraints(opts); len(ands) > 0 {
		sql += " and " + strings.Join(ands, " and ")
	}
	sql += ` order by score desc` + d.offset(opts)

	return d.queryEntries(sql, query)
}

func (d *Postgres) whereConstraints(opts *QueryOptions) []string {
	var where []string

	if !opts.Deleted {
		where = append(where, "isDeleted=false")
	}

	if !opts.Private {
		where = append(where, "isPrivate=false")
	}

	if !opts.Draft {
		where = append(where, "isDraft=false")
	}

	return where
}

func (d *Postgres) offset(opts *QueryOptions) string {
	var sql string

	if opts.Page > 0 {
		sql += " offset " + strconv.Itoa(opts.Page*opts.Limit)
	}

	return sql + " limit " + strconv.Itoa(opts.Limit)
}

func (d *Postgres) queryEntries(sql string, args ...interface{}) ([]string, error) {
	rows, err := d.pool.Query(context.Background(), sql, args...)
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

	return ids, nil
}

func (d *Postgres) Close() {
	d.pool.Close()
}
