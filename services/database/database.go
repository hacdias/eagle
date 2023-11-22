package database

import (
	"bytes"
	"context"
	"encoding/binary"
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

func (b *Database) AddMention(ctx context.Context, mention *core.Mention) error {
	mention.ID = uuid.New().String()
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(mention)
	if err != nil {
		return err
	}

	return b.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("mentions"))
		if err != nil {
			return err
		}

		return b.Put([]byte(mention.ID), buf.Bytes())
	})
}

func (b *Database) GetMentions(ctx context.Context) ([]*core.Mention, error) {
	var mentions []*core.Mention

	return mentions, b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("mentions"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var mention *core.Mention
			err := gob.NewDecoder(bytes.NewReader(v)).Decode(&mention)
			if err != nil {
				return err
			}

			mentions = append(mentions, mention)
			return nil
		})
	})
}

func (b *Database) GetMention(ctx context.Context, id string) (*core.Mention, error) {
	var mention *core.Mention

	return mention, b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("mentions"))
		if b == nil {
			return errors.New("mention does not exist")
		}

		v := b.Get([]byte(id))
		if v == nil {
			return errors.New("mention does not exist")
		}

		return gob.NewDecoder(bytes.NewReader(v)).Decode(&mention)
	})
}

func (b *Database) DeleteMention(ctx context.Context, id string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("mentions"))
		if b == nil {
			return nil
		}

		return b.Delete([]byte(id))
	})
}

func (b *Database) AddTaxonomy(ctx context.Context, taxonomy string, taxons ...string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("taxonomies"))
		if err != nil {
			return err
		}

		b, err = b.CreateBucketIfNotExists([]byte(taxonomy))
		if err != nil {
			return err
		}

		for _, taxon := range taxons {
			v := b.Get([]byte(taxon))
			if v == nil {
				err = b.Put([]byte(taxon), binary.AppendVarint(nil, 1))
			} else {
				count, n := binary.Varint(v)
				if n <= 0 {
					return errors.New("could not read")
				}

				err = b.Put([]byte(taxon), binary.AppendVarint(nil, count+1))
			}

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (b *Database) GetTaxonomy(ctx context.Context, taxonomy string) ([]string, error) {
	var taxons []string

	return taxons, b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("taxonomies"))
		if b == nil {
			return nil
		}

		b = b.Bucket([]byte(taxonomy))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			count, n := binary.Varint(v)
			if n <= 0 {
				return errors.New("could not read")
			}

			if count > 0 {
				taxons = append(taxons, string(k))
			}
		}
		return nil
	})
}

func (b *Database) DeleteTaxonomy(ctx context.Context, taxonomy string, taxons ...string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("taxonomies"))
		if err != nil {
			return err
		}

		b, err = b.CreateBucketIfNotExists([]byte(taxonomy))
		if err != nil {
			return err
		}

		for _, taxon := range taxons {
			v := b.Get([]byte(taxon))
			if v != nil {
				count, n := binary.Varint(v)
				if n <= 0 {
					return errors.New("could not read")
				}

				count = count - 1
				if count <= 0 {
					err = b.Delete([]byte(taxon))
				} else {
					err = b.Put([]byte(taxon), binary.AppendVarint(nil, count))
				}

				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (b *Database) ResetTaxonomies(ctx context.Context) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte("taxonomies"))
		if errors.Is(err, bolt.ErrBucketNotFound) {
			return nil
		}
		return err
	})
}
