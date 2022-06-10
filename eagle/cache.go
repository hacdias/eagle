package eagle

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hacdias/eagle/v4/entry"
	"github.com/hacdias/eagle/v4/entry/mf2"
)

type CacheScope string

const (
	CacheRegular CacheScope = "reg"
	CacheTor     CacheScope = "tor"
)

func (e *Eagle) initCache() error {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000 * 10,        // 1000 items when full with 30 KB items -> x10
		MaxCost:     30 * 1000 * 1000, // 30 MB
		BufferItems: 64,               // recommended value
	})
	if err != nil {
		return err
	}
	e.cache = cache
	return nil
}

type cacheEntry struct {
	data    []byte
	modtime time.Time
}

func (e *Eagle) SaveCache(scope CacheScope, filename string, data []byte, modtime time.Time) {
	value := &cacheEntry{data, modtime}
	cost := int64(len(data))
	e.cache.SetWithTTL(e.cacheKey(scope, filename), value, cost, time.Hour*24)
}

func (e *Eagle) RemoveCache(ee *entry.Entry) {
	e.PurgeCache("/")
	e.PurgeCache("/all")
	e.PurgeCache(ee.ID)

	for _, sec := range ee.Sections {
		e.PurgeCache("/" + sec)
	}

	hasTags := false
	for _, tag := range ee.Tags() {
		hasTags = true
		e.PurgeCache("/tag/" + tag)
	}
	if hasTags {
		e.PurgeCache("/tags")
	}

	hasEmojis := false
	for _, emoji := range ee.Emojis() {
		hasEmojis = true
		e.PurgeCache("/emoji/" + emoji)
	}
	if hasEmojis {
		e.PurgeCache("/emojis")
	}

	if mm := ee.Helper(); mm.PostType() == mf2.TypeRead {
		canonical := mm.String(mm.TypeProperty())
		if canonical != "" {
			e.PurgeCache(canonical)
		}
	}
}

func (e *Eagle) PurgeCache(filename string) {
	e.cache.Del(e.cacheKey(CacheRegular, filename))
	e.cache.Del(e.cacheKey(CacheTor, filename))
}

func (e *Eagle) ResetCache() {
	e.cache.Clear()
}

func (e *Eagle) IsCached(scope CacheScope, filename string) ([]byte, time.Time, bool) {
	data, ok := e.cache.Get(e.cacheKey(scope, filename))
	if ok {
		ce := data.(*cacheEntry)
		return ce.data, ce.modtime, true
	}
	return nil, time.Time{}, false
}

func (e *Eagle) cacheKey(scope CacheScope, filename string) string {
	return string(scope) + filename
}
