package watcher

import (
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

type Loader interface {
	Load(path string) error
}

type Watcher struct {
	stop chan struct{}
	done chan error
}

func LoadAndWatch(path string, loader Loader) (*Watcher, error) {
	err := loader.Load(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load file")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create watcher")
	}
	err = watcher.Add(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add file to watcher")
	}
	stop := make(chan struct{})
	done := make(chan error)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					err := loader.Load(path)
					if err != nil {
						log.Println(errors.Wrap(err, "failed to reload cms"))
					}
				}
			case err := <-watcher.Errors:
				log.Println(errors.Wrap(err, "failed to watch cms"))
			case <-stop:
				done <- watcher.Close()
				return
			}
		}
	}()
	return &Watcher{stop: stop, done: done}, nil
}

func (w *Watcher) Close() error {
	close(w.stop)
	return <-w.done
}
