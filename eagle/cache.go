package eagle

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/hacdias/eagle/v2/entry"
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

func (e *Eagle) SaveCache(filename string, data []byte, modtime time.Time) {
	value := &cacheEntry{data, modtime}
	cost := int64(len(data))
	e.cache.SetWithTTL(filename, value, cost, time.Hour*24)
}

func (e *Eagle) RemoveCache(ee *entry.Entry) {
	e.cache.Del("/")
	e.cache.Del(ee.ID)

	for _, sec := range ee.Sections {
		e.cache.Del("/" + sec)
	}

	hasTags := false
	for _, tag := range ee.Tags() {
		hasTags = true
		e.cache.Del("/tag/" + tag)
	}
	if hasTags {
		e.cache.Del("/tags")
	}
}

func (e *Eagle) ResetCache() {
	e.cache.Clear()
}

func (e *Eagle) IsCached(filename string) ([]byte, time.Time, bool) {
	data, ok := e.cache.Get(filename)
	if ok {
		ce := data.(*cacheEntry)
		return ce.data, ce.modtime, true
	}
	return nil, time.Time{}, false
}
