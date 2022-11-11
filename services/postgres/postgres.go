package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hacdias/eagle/eagle"
	"github.com/hacdias/eagle/hooks"
	"github.com/hacdias/eagle/indexer"
	"github.com/hacdias/eagle/util"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(cfg *eagle.PostgreSQL) (*Postgres, error) {
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

func (d *Postgres) Add(entries ...*eagle.Entry) error {
	b := &pgx.Batch{}

	for _, entry := range entries {
		content := entry.Title + " " + entry.Description + " " + entry.TextContent() + " " + strings.Join(entry.Tags(), " ")

		updated := entry.Published.UTC()
		if !entry.Updated.IsZero() {
			updated = entry.Updated.UTC()
		}

		b.Queue("delete from entries where id=$1", entry.ID)
		b.Queue("insert into entries(id, content, isDraft, isDeleted, visibility, audience, date, updated, properties) values($1, $2, $3, $4, $5, $6, $7, $8, $9)",
			entry.ID, content, entry.Draft, entry.Deleted, entry.Visibility(), entry.Audience(), entry.Published.UTC(), updated, entry.Properties)

		for _, tag := range entry.Tags() {
			b.Queue("insert into tags(entry_id, tag) values ($1, $2)", entry.ID, util.Slugify(tag))
		}

		for _, emoji := range entry.Emojis() {
			b.Queue("insert into emojis(entry_id, emoji) values ($1, $2)", entry.ID, emoji)
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

func (d *Postgres) GetEmojis() ([]string, error) {
	rows, err := d.pool.Query(context.Background(), "select emoji, count(*) from emojis group by emoji order by count desc")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []string{}

	for rows.Next() {
		var id string
		var count int
		err := rows.Scan(&id, &count)
		if err != nil {
			return nil, err
		}
		tags = append(tags, id)
	}

	return tags, rows.Err()
}

// TODO: support opts.OrderByUpdated?
func (d *Postgres) ByDate(opts *indexer.Query, year, month, day int) ([]string, error) {
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

	wwhere, aargs := d.whereConstraints(opts, i)

	where = append(where, wwhere...)
	args = append(args, aargs...)
	sql += strings.Join(where, " and ")
	sql += " order by date desc"
	sql += d.offset(opts.Pagination)

	return d.queryEntries(sql, 0, args...)
}

// TODO: support opts.OrderByUpdated?
func (d *Postgres) ByTag(opts *indexer.Query, tag string) ([]string, error) {
	args := []interface{}{tag}
	sql := "select id from tags inner join entries on id=entry_id where tag=$1"

	if where, aargs := d.whereConstraints(opts, 1); len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
		args = append(args, aargs...)
	}

	sql += " order by date desc" + d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

// TODO: support opts.OrderByUpdated?
func (d *Postgres) ByEmoji(opts *indexer.Query, emoji string) ([]string, error) {
	args := []interface{}{emoji}
	sql := "select id from emojis inner join entries on id=entry_id where emoji=$1"

	if where, aargs := d.whereConstraints(opts, 1); len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
		args = append(args, aargs...)
	}

	sql += " order by date desc" + d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

func (d *Postgres) BySection(opts *indexer.Query, sections ...string) ([]string, error) {
	sql := "select distinct (id) id, updated, date from sections inner join entries on id=entry_id"
	where, args := d.whereConstraints(opts, 0)
	i := len(args)

	var sectionsWhere []string
	for _, section := range sections {
		i++
		sectionsWhere = append(sectionsWhere, "section=$"+strconv.Itoa(i))
		args = append(args, section)
	}

	if len(sectionsWhere) > 0 {
		where = append(where, "("+strings.Join(sectionsWhere, " or ")+")")
	}

	if len(where) > 0 {
		sql += " where " + strings.Join(where, " and ")
	}

	if opts.OrderByUpdated {
		sql += " order by updated desc"
	} else {
		sql += " order by date desc"
	}

	sql += d.offset(opts.Pagination)
	return d.queryEntries(sql, 2, args...)
}

// TODO: support opts.OrderByUpdated?
func (d *Postgres) ByProperty(opts *indexer.Query, property, value string) ([]string, error) {
	args := []interface{}{property, value}
	sql := "select id from entries where properties->>$1=$2"

	if where, aargs := d.whereConstraints(opts, 2); len(where) > 0 {
		sql += " and " + strings.Join(where, " and ")
		args = append(args, aargs...)
	}

	sql += " order by date desc" + d.offset(opts.Pagination)
	return d.queryEntries(sql, 0, args...)
}

func (d *Postgres) GetAll(opts *indexer.Query) ([]string, error) {
	sql := "select id from entries"

	where, args := d.whereConstraints(opts, 0)
	if len(where) > 0 {
		sql += " where " + strings.Join(where, " and ")
	}

	return d.queryEntries(sql+" order by date desc"+d.offset(opts.Pagination), 0, args...)
}

func (d *Postgres) GetDeleted(opts *indexer.Pagination) ([]string, error) {
	sql := "select id from entries where isDeleted=true order by date desc" + d.offset(opts)
	return d.queryEntries(sql, 0)
}

func (d *Postgres) GetDrafts(opts *indexer.Pagination) ([]string, error) {
	sql := "select id from entries where isDraft=true order by date desc" + d.offset(opts)
	return d.queryEntries(sql, 0)
}

func (d *Postgres) GetUnlisted(opts *indexer.Pagination) ([]string, error) {
	sql := "select id from entries where visibility='unlisted' order by date desc" + d.offset(opts)
	return d.queryEntries(sql, 0)
}

func (d *Postgres) GetPrivate(opts *indexer.Pagination, audience string) ([]string, error) {
	if audience == "" {
		return nil, errors.New("audience is required")
	}

	sql := "select id from entries where visibility='private' and $1=any(audience) and isDraft=false and isDeleted=false order by date desc" + d.offset(opts)
	return d.queryEntries(sql, 0, audience)
}

func (d *Postgres) Search(opts *indexer.Query, search *indexer.Search) ([]string, error) {
	mainSelect := "select ts_rank_cd(ts, plainto_tsquery('english', $1)) as score, id, isDraft, isDeleted, visibility, audience"
	mainFrom := "from entries as e"

	if len(search.Sections) > 0 {
		mainSelect += ", section"
		mainFrom += " inner join sections on e.id = sections.entry_id"
	}

	if len(search.Tags) > 0 {
		mainSelect += ", tag"
		mainFrom += " inner join tags on e.id = tags.entry_id"
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
		for i, section := range search.Sections {
			sectionsSql = append(sectionsSql, "section=$"+strconv.Itoa(i+2))
			args = append(args, section)
		}

		sql += " and (" + strings.Join(sectionsSql, " or ") + ")"
	}

	if len(search.Tags) > 0 {
		tagsSql := []string{}
		for i, section := range search.Tags {
			tagsSql = append(tagsSql, "tag=$"+strconv.Itoa(i+2+len(search.Sections)))
			args = append(args, section)
		}

		sql += " and (" + strings.Join(tagsSql, " or ") + ")"
	}

	sql += ` order by score desc` + d.offset(opts.Pagination)
	return d.queryEntries(sql, 1, args...)
}

func (d *Postgres) GetReadsSummary() (*hooks.ReadsSummary, error) {
	sql := `select distinct on (id)
	id,
	updated as date,
	properties->'read-status'->0->>'status' as status,
 	properties->'read-of'->'properties'->>'name' as name,
	properties->'read-of'->'properties'->>'author' as author
from entries
where properties->'read-status'->0->>'status' is not null`

	if ands, _ := d.whereConstraints(&indexer.Query{}, 0); len(ands) > 0 {
		sql += " and " + strings.Join(ands, " and ")
	}

	sql += " order by id, name, date desc"

	rows, err := d.pool.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &hooks.ReadsSummary{
		ToRead:  []*hooks.Read{},
		Reading: []*hooks.Read{},
	}

	finished := hooks.ReadList([]*hooks.Read{})

	for rows.Next() {
		read := &hooks.Read{}
		status := ""

		err := rows.Scan(&read.ID, &read.Date, &status, &read.Name, &read.Author)
		if err != nil {
			return nil, err
		}

		switch status {
		case "to-read":
			stats.ToRead = append(stats.ToRead, read)
		case "reading":
			stats.Reading = append(stats.Reading, read)
		case "finished":
			finished = append(finished, read)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats.Finished = *finished.ByYear()
	return stats, nil
}

func (d *Postgres) watches(baseSql string) ([]*hooks.Watch, error) {
	sql := "select id, date, name from (" + baseSql

	if ands, _ := d.whereConstraints(&indexer.Query{}, 0); len(ands) > 0 {
		sql += " and " + strings.Join(ands, " and ")
	}

	sql += " order by ttid, date desc ) s order by date desc"

	rows, err := d.pool.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	watches := []*hooks.Watch{}

	for rows.Next() {
		watch := &hooks.Watch{}
		err := rows.Scan(&watch.ID, &watch.Date, &watch.Name)
		if err != nil {
			return nil, err
		}
		watches = append(watches, watch)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return watches, nil
}

func (d *Postgres) GetWatchesSummary() (*hooks.WatchesSummary, error) {
	watches := &hooks.WatchesSummary{
		Series: []*hooks.Watch{},
		Movies: []*hooks.Watch{},
	}

	series, err := d.watches(`select distinct on (ttid)
	id,
	date,
	properties->'watch-of'->'properties'->'episode-of'->'properties'->>'name' as name,
	properties->'watch-of'->'properties'->'episode-of'->'properties'->'trakt-ids'->>'trakt' as ttid
from entries
where
	properties->'watch-of'->'properties'->'episode-of' is not null`)
	if err != nil {
		return nil, err
	}
	watches.Series = series

	movies, err := d.watches(`select distinct on (ttid)
	id,
	date,
	properties->'watch-of'->'properties'->>'name' as name,
	properties->'watch-of'->'properties'->'trakt-ids'->>'trakt' as ttid
from entries
where
	properties->'watch-of' is not null and
	properties->'watch-of'->'properties'->'episode-of' is null and
	properties->'watch-of'->'properties'->'trakt-ids'->>'trakt' is not null`)
	if err != nil {
		return nil, err
	}
	watches.Movies = movies

	return watches, nil
}

func (d *Postgres) whereConstraints(opts *indexer.Query, i int) ([]string, []interface{}) {
	var where []string
	var args []interface{}

	if !opts.WithDeleted {
		where = append(where, "isDeleted=false")
	}

	if len(opts.Visibility) > 0 {
		visibilityOr := []string{}
		for _, vis := range opts.Visibility {
			i++
			if vis == eagle.VisibilityPrivate && opts.Audience != "" {
				visibilityOr = append(visibilityOr, "(visibility='private' and audience is null)")
				visibilityOr = append(visibilityOr, "(visibility='private' and $"+strconv.Itoa(i)+" = ANY (audience) )")
				args = append(args, opts.Audience)
			} else {
				visibilityOr = append(visibilityOr, "visibility=$"+strconv.Itoa(i))
				args = append(args, vis)
			}
		}
		where = append(where, "("+strings.Join(visibilityOr, " or ")+")")
	}

	if !opts.WithDrafts {
		where = append(where, "isDraft=false")
	}

	return where, args
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

func (d *Postgres) Close() error {
	d.pool.Close()
	return nil
}

func (d *Postgres) Been() ([]string, error) {
	sql := `select distinct on (country)
	properties->'location'->'properties'->>'country-name' as country
from entries
where properties->'location'->'properties'->>'country-name' is not null
order by country`

	rows, err := d.pool.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var been []string

	for rows.Next() {
		var country string
		err := rows.Scan(&country)
		if err != nil {
			return nil, err
		}

		been = append(been, country)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return been, nil
}

func (d *Postgres) SectionsCount() (map[string]int, error) {
	sql := `select section, COUNT(*)
	from sections inner join entries on id=entry_id
	group by section
	order by section;`

	rows, err := d.pool.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	count := map[string]int{}

	for rows.Next() {
		var (
			section string
			n       int
		)
		err := rows.Scan(&section, &n)
		if err != nil {
			return nil, err
		}

		count[section] = n
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return count, nil
}