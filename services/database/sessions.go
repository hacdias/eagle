package database

import (
	"bytes"
	"context"
	"encoding/gob"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Session represents a login session.
type Session struct {
	ID      string
	Expiry  time.Time
	Created time.Time
}

func (b *Database) AddSession(ctx context.Context, session *Session) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(session); err != nil {
		return err
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(session.ID), buf.Bytes())
	})
}

func (b *Database) GetSession(ctx context.Context, id string) (*Session, error) {
	var session Session
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("sessions"))
		if bucket == nil {
			return ErrNotFound
		}

		v := bucket.Get([]byte(id))
		if v == nil {
			return ErrNotFound
		}
		return gob.NewDecoder(bytes.NewReader(v)).Decode(&session)
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (b *Database) DeleteSession(ctx context.Context, id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("sessions"))
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(id))
	})
}

func (b *Database) GetSessions(ctx context.Context) ([]*Session, error) {
	var sessions []*Session
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("sessions"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var session Session
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&session); err != nil {
				return err
			}
			sessions = append(sessions, &session)
			return nil
		})
	})

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Created.Before(sessions[j].Created)
	})

	return sessions, err
}

func (b *Database) DeleteAllSessions(ctx context.Context) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("sessions")) == nil {
			return nil
		}

		return tx.DeleteBucket([]byte("sessions"))
	})
}

func (b *Database) DeleteExpiredSessions(ctx context.Context) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("sessions"))
		if bucket == nil {
			return nil
		}

		now := time.Now()
		var toDelete []string
		err := bucket.ForEach(func(k, v []byte) error {
			var session Session
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&session); err != nil {
				return nil
			}

			if !session.Expiry.IsZero() && now.After(session.Expiry) {
				toDelete = append(toDelete, string(k))
			}

			return nil
		})
		if err != nil {
			return err
		}

		for _, id := range toDelete {
			if err := bucket.Delete([]byte(id)); err != nil {
				return err
			}
		}
		return nil
	})
}
