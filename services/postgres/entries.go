package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/indexer"
	"github.com/jackc/pgx/v5"
)

func (d *Postgres) Add(entries ...*eagle.Entry) error {
	b := &pgx.Batch{}

	for _, entry := range entries {
		content := entry.Title + " " + entry.Description + " " + entry.TextContent()

		updated := entry.Published.UTC()
		if !entry.Updated.IsZero() {
			updated = entry.Updated.UTC()
		}

		b.Queue("delete from entries where id=$1", entry.ID)
		b.Queue("insert into entries(id, content, isDraft, isDeleted, isUnlisted, published_at, updated_at) values($1, $2, $3, $4, $5, $6, $7)",
			entry.ID, content, entry.Draft, entry.Deleted, entry.Unlisted, entry.Published.UTC(), updated)

		for taxonomy, terms := range entry.Taxonomies {
			for _, term := range terms {
				b.Queue("insert into taxonomies(entry_id, taxonomy, term) values ($1, $2, $3)", entry.ID, taxonomy, term)
			}
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

func (d *Postgres) Remove(ids ...string) {
	for _, id := range ids {
		_, _ = d.pool.Exec(context.Background(), "delete from entries where id=$1", id)
	}
}

func (d *Postgres) GetAll(opts *indexer.Query) ([]string, error) {
	sql := "select id from entries"

	where, args := d.whereConstraints(opts, 0)
	if len(where) > 0 {
		sql += " where " + strings.Join(where, " and ")
	}

	sql += d.orderBy(opts)
	sql += d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

func (d *Postgres) GetDrafts(opts *indexer.Pagination) ([]string, error) {
	sql := "select id from entries where isDraft=true order by published_at desc" + d.offset(opts)
	return d.queryEntries(sql, 0)
}

func (d *Postgres) GetUnlisted(opts *indexer.Pagination) ([]string, error) {
	sql := "select id from entries where isUnlisted=true order by published_at desc" + d.offset(opts)
	return d.queryEntries(sql, 0)
}

func (d *Postgres) GetDeleted(opts *indexer.Pagination) ([]string, error) {
	sql := "select id from entries where isDeleted=true order by published_at desc" + d.offset(opts)
	return d.queryEntries(sql, 0)
}

func (d *Postgres) GetBySection(opts *indexer.Query, section string) ([]string, error) {
	args := []interface{}{section}
	sql := "select id from entries inner join sections on id=entry_id where section=$1"

	if where, aargs := d.whereConstraints(opts, 1); len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
		args = append(args, aargs...)
	}

	sql += d.orderBy(opts)
	sql += d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

func (d *Postgres) GetByTaxonomy(opts *indexer.Query, taxonomy, term string) ([]string, error) {
	args := []interface{}{taxonomy, term}
	sql := "select id from entries inner join taxonomies on id=entry_id where taxonomy=$1 and term=$2"

	if where, aargs := d.whereConstraints(opts, 2); len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
		args = append(args, aargs...)
	}

	sql += d.orderBy(opts)
	sql += d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

func (d *Postgres) GetByDate(opts *indexer.Query, year, month, day int) ([]string, error) {
	if year == 0 && month == 0 && day == 0 {
		return nil, errors.New("year, month or day must be set")
	}

	sql := "select id from entries where "
	args := []interface{}{}
	where := []string{}

	if year > 0 {
		args = append(args, year)
		where = append(where, "date_part('year', published_at)=$"+strconv.Itoa(len(args)))
	}

	if month > 0 {
		args = append(args, month)
		where = append(where, "date_part('month', published_at)=$"+strconv.Itoa(len(args)))
	}

	if day > 0 {
		args = append(args, day)
		where = append(where, "date_part('day', published_at)=$"+strconv.Itoa(len(args)))
	}

	if w, a := d.whereConstraints(opts, len(args)); len(w) > 0 {
		where = append(where, w...)
		args = append(args, a...)
	}

	sql += strings.Join(where, " and ")
	sql += d.orderBy(opts)
	sql += d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

func (d *Postgres) GetTaxonomyTerms(taxonomy string) (eagle.Terms, error) {
	rows, err := d.pool.Query(context.Background(), "select distinct (term) term from taxonomies where taxonomy=$1 order by term", taxonomy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := []string{}
	for rows.Next() {
		var term string
		err := rows.Scan(&term)
		if err != nil {
			return nil, err
		}
		terms = append(terms, term)
	}

	return terms, nil
}

func (d *Postgres) GetSearch(opts *indexer.Query, search *indexer.Search) ([]string, error) {
	mainSelect := "select ts_rank_cd(ts, plainto_tsquery('english', $1)) as score, id, isDraft, isDeleted, isUnlisted"
	mainFrom := "from entries as e"

	if len(search.Sections) > 0 {
		mainSelect += ", section"
		mainFrom += " inner join sections on e.id = sections.entry_id"
	}

	sql := "select distinct (id) id, score from (" + mainSelect + " " + mainFrom + ") s where score > 0"

	args := []interface{}{search.Query}
	where, wargs := d.whereConstraints(opts, 1)
	args = append(args, wargs...)

	if len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
	}

	if len(search.Sections) > 0 {
		sectionsSql := []string{}
		for _, section := range search.Sections {
			args = append(args, section)
			sectionsSql = append(sectionsSql, "section=$"+strconv.Itoa(len(args)))
		}

		sql += " and (" + strings.Join(sectionsSql, " or ") + ")"
	}

	sql += ` order by score desc` + d.offset(opts.Pagination)
	return d.queryEntries(sql, 1, args...)
}

func (d *Postgres) GetCount() (int, error) {
	sql := `select count(*) from entries;`

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

func (d *Postgres) ClearEntries() {
	_, _ = d.pool.Exec(context.Background(), "truncate table entries cascade")
}

func (d *Postgres) whereConstraints(opts *indexer.Query, i int) ([]string, []interface{}) {
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

	if !opts.After.IsZero() {
		i++
		where = append(where, "published_at>=$"+strconv.Itoa(i))
		args = append(args, opts.After)
	}

	if !opts.Before.IsZero() {
		i++
		where = append(where, "published_at<=$"+strconv.Itoa(i))
		args = append(args, opts.Before)
	}

	return where, args
}

func (d *Postgres) orderBy(opts *indexer.Query) string {
	q := ""

	if opts.OrderByUpdated {
		q = " order by updated_at "
	} else {
		q = " order by published_at "
	}

	if opts.Ascending {
		return q + "asc"
	}

	return q + "desc"
}

func (d *Postgres) offset(opts *indexer.Pagination) string {
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
