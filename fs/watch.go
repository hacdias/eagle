package fs

import (
	"io/fs"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hacdias/eagle/log"
)

func (f *FS) Watch(dir string, exec func() error) {
	log := log.S().Named("watcher")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case evt, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Ignore CHMOD only events.
				if evt.Op != fsnotify.Chmod {
					log.Infof("%s changed", evt.Name)
					err := exec()
					if err != nil {
						log.Error(err)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Error(err)
			}
		}
	}()

	err = f.Afero.Walk(dir, func(filename string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		return watcher.Add(filepath.Join(f.path, filename))
	})
	if err != nil {
		log.Error(err)
		return
	}

	<-make(chan struct{})
}
