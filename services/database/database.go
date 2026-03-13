package database

import (
	"errors"

	"go.hacdias.com/eagle/core"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Database struct {
	db *gorm.DB
}

func NewDatabase(path string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&Session{}, &Token{}, &core.Mention{})
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
