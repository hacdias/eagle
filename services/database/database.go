package database

import (
	"go.hacdias.com/eagle/core"
	"go.hacdias.com/eagle/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	db *gorm.DB
}

func NewDatabase(path string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: log.NewGormLogger(),
	})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&core.Token{}, &core.Mention{})
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
