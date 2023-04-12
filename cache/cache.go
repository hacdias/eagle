package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hacdias/eagle/eagle"
)

type Cache struct {
	r *ristretto.Cache
}

func NewCache() (*Cache, error) {
	r, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000 * 10,        // 1000 items when full with 30 KB items -> x10
		MaxCost:     30 * 1000 * 1000, // 30 MB
		BufferItems: 64,               // recommended value
	})
	if err != nil {
		return nil, err
	}

	return &Cache{
		r: r,
	}, nil
}

type cacheEntry struct {
	data    []byte
	modtime time.Time
}

func (c *Cache) Save(filename string, data []byte, modtime time.Time) {
	value := &cacheEntry{data, modtime}
	cost := int64(len(data))
	c.r.SetWithTTL(filename, value, cost, time.Hour*24)
}

func (c *Cache) Delete(ee *eagle.Entry) {
	c.delete("/")
	c.delete(ee.ID)
}

func (c *Cache) Clear() {
	c.r.Clear()
}

func (c *Cache) Cached(filename string) ([]byte, time.Time, bool) {
	data, ok := c.r.Get(filename)
	if ok {
		ce := data.(*cacheEntry)
		return ce.data, ce.modtime, true
	}
	return nil, time.Time{}, false
}

func (c *Cache) delete(filename string) {
	c.r.Del(filename)
}
