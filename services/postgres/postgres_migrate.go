package postgres

import (
	"database/sql"
	"embed"
	"path/filepath"
	"sort"

	"github.com/hacdias/eagle/log"
	_ "github.com/jackc/pgx/v4/stdlib" // postgres driver
	"github.com/lopezator/migrator"
)

//go:embed migrations/*
var migrationsFs embed.FS

func migrate(dsn string) error {
	// Open database connection
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	m, err := migrator.New(
		migrator.WithLogger(migrator.LoggerFunc(log.S().Named("migration").Infof)),
		migrator.Migrations(migrations...),
	)
	if err != nil {
		return err
	}

	return m.Migrate(db)
}

func loadMigrations() ([]interface{}, error) {
	files, err := migrationsFs.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	filenames := []string{}
	for _, file := range files {
		if file.Type().IsDir() {
			continue
		}

		filenames = append(filenames, filepath.Join("migrations", file.Name()))
	}

	sort.Strings(filenames)

	var migrations []interface{}

	for _, filename := range filenames {
		fd, err := migrationsFs.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		if len(fd) == 0 {
			continue
		}

		mig := &migrator.Migration{}
		mig.Name = filepath.Base(filename)
		mig.Func = func(t *sql.Tx) error {
			_, txe := t.Exec(string(fd))
			return txe
		}

		migrations = append(migrations, mig)
	}

	return migrations, err
}
