package database

import (
	"bytes"
	"context"
	"encoding/gob"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Token represents an IndieAuth JWT token persisted in the database.
type Token struct {
	ID       string
	ClientID string
	Scope    string
	Expiry   time.Time // zero means no expiry
	Created  time.Time
}

func (b *Database) AddToken(ctx context.Context, token *Token) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(token); err != nil {
		return err
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("tokens"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(token.ID), buf.Bytes())
	})
}

func (b *Database) HasToken(ctx context.Context, id string) (bool, error) {
	var found bool
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		if bucket == nil {
			return nil
		}

		found = bucket.Get([]byte(id)) != nil
		return nil
	})
	return found, err
}

func (b *Database) DeleteToken(ctx context.Context, id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(id))
	})
}

func (b *Database) GetTokens(ctx context.Context) ([]*Token, error) {
	var tokens []*Token

	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			var token Token
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&token); err != nil {
				return err
			}
			tokens = append(tokens, &token)
			return nil
		})
	})

	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].Created.Before(tokens[j].Created)
	})

	return tokens, err
}

func (b *Database) DeleteExpiredTokens(ctx context.Context) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("tokens"))
		if bucket == nil {
			return nil
		}

		now := time.Now()
		var toDelete []string

		err := bucket.ForEach(func(k, v []byte) error {
			var token Token
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&token); err != nil {
				return nil
			}

			if !token.Expiry.IsZero() && now.After(token.Expiry) {
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

func (b *Database) DeleteAllTokens(ctx context.Context) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("tokens")) == nil {
			return nil
		}
		return tx.DeleteBucket([]byte("tokens"))
	})
}
