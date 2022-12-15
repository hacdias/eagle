package cache

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hacdias/eagle/eagle"
)

type CacheScope string

const (
	CacheRegular CacheScope = "reg"
	CacheTor     CacheScope = "tor"
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

func (c *Cache) Save(scope CacheScope, filename string, data []byte, modtime time.Time) {
	value := &cacheEntry{data, modtime}
	cost := int64(len(data))
	c.r.SetWithTTL(c.cacheKey(scope, filename), value, cost, time.Hour*24)
}

func (c *Cache) Delete(ee *eagle.Entry) {
	c.delete("/")
	c.delete("/all")
	c.delete(ee.ID)

	for _, section := range ee.Sections {
		c.delete("/" + section)
	}

	for taxonomy, terms := range ee.Taxonomies {
		for _, term := range terms {
			c.delete("/" + taxonomy + "/" + term)
		}
		c.delete("/" + taxonomy)
	}

	// TODO: invalidate year/month/day archives.
}

func (c *Cache) Clear() {
	c.r.Clear()
}

func (c *Cache) Cached(scope CacheScope, filename string) ([]byte, time.Time, bool) {
	data, ok := c.r.Get(c.cacheKey(scope, filename))
	if ok {
		ce := data.(*cacheEntry)
		return ce.data, ce.modtime, true
	}
	return nil, time.Time{}, false
}

func (c *Cache) cacheKey(scope CacheScope, filename string) string {
	return string(scope) + filename
}

func (c *Cache) delete(filename string) {
	c.r.Del(c.cacheKey(CacheRegular, filename))
	c.r.Del(c.cacheKey(CacheTor, filename))
}
