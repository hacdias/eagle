package activitypub

import (
	"encoding/json"
	"os"
	"sync"
)

type stringMapStore struct {
	sync.RWMutex
	data map[string]string
	file string
}

func newStringMapStore(file string) (*stringMapStore, error) {
	fm := &stringMapStore{
		data: map[string]string{},
		file: file,
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fm, fm.save()
	} else if err != nil {
		return nil, err
	}

	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(bytes, &fm.data)
	if err != nil {
		return nil, err
	}

	return fm, nil
}

func (f *stringMapStore) getAll() map[string]string {
	f.RLock()
	defer f.RUnlock()

	m := make(map[string]string)
	for key, value := range f.data {
		m[key] = value
	}

	return m
}

func (f *stringMapStore) get(key string) (string, bool) {
	f.RLock()
	v, ok := f.data[key]
	f.RUnlock()
	return v, ok
}

func (f *stringMapStore) remove(key string) error {
	f.Lock()
	defer f.Unlock()
	delete(f.data, key)
	return f.save()
}

func (f *stringMapStore) set(key, value string) error {
	f.Lock()
	defer f.Unlock()
	f.data[key] = value
	return f.save()
}

func (f *stringMapStore) save() error {
	bytes, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.file, bytes, 0644)
}
