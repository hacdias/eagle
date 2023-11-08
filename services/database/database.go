package database

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
	"go.hacdias.com/eagle/core"
)

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

func (b *Database) AddGuestbookEntry(ctx context.Context, entry *core.GuestbookEntry) error {
	entry.ID = uuid.New().String()
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(entry)
	if err != nil {
		return err
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("guestbook"))
		if err != nil {
			return err
		}

		return b.Put([]byte(entry.ID), buf.Bytes())
	})
}

func (b *Database) GetGuestbookEntries(ctx context.Context) (core.GuestbookEntries, error) {
	var ee core.GuestbookEntries

	return ee, b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("guestbook"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var e core.GuestbookEntry
			err := gob.NewDecoder(bytes.NewReader(v)).Decode(&e)
			if err != nil {
				return err
			}

			ee = append(ee, e)
			return nil
		})
	})
}

func (b *Database) GetGuestbookEntry(ctx context.Context, id string) (core.GuestbookEntry, error) {
	var e core.GuestbookEntry

	return e, b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("guestbook"))
		if b == nil {
			return errors.New("entry doe not exist")
		}

		v := b.Get([]byte(id))
		if v == nil {
			return errors.New("entry doe not exist")
		}

		return gob.NewDecoder(bytes.NewReader(v)).Decode(&e)
	})
}

func (b *Database) DeleteGuestbookEntry(ctx context.Context, id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("guestbook"))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(id))
	})
}
