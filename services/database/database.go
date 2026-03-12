package database

import (
	"errors"

	bolt "go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("not found")

type Database struct {
	db *bolt.DB
}

func NewDatabase(path string) (*Database, error) {
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

func (b *Database) Close() error {
	return b.db.Close()
}
