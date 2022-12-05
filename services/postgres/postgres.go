package postgres

import (
	"context"

	"github.com/hacdias/eagle/eagle"
	"github.com/jackc/pgx/v5/pgxpool"
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

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	return &Postgres{pool}, nil
}

func (d *Postgres) Close() error {
	d.pool.Close()
	return nil
}
