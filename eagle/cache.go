package eagle

import (
	"path/filepath"
)

func (e *Eagle) SaveCache(filename string, data []byte) error {
	err := e.cacheFs.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		return err
	}

	return e.cacheFs.WriteFile(filename, data, 0644)
}

func (e *Eagle) RemoveCache(filename string) error {
	return e.cacheFs.RemoveAll(filename)
}

func (e *Eagle) ResetCache() error {
	return e.cacheFs.RemoveAll(".")
}

func (e *Eagle) IsCached(filename string) (string, bool) {
	stat, err := e.cacheFs.Stat(filename)
	return filepath.Join(e.Config.CacheDirectory, filename), err == nil && stat.Mode().IsRegular()
}
