package daemon

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher is a wrapper around fsnotify.Watcher
// It is easier to use and manage errors locally
// It also avoids multiple events sent at each change
// Note that it create a goroutine to preprocess events
type Watcher struct {
	watcher *fsnotify.Watcher
	Events  chan fsnotify.Event
}

func NewWatcher() *Watcher {
	internalWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Error creating watcher: ", err)
		log.Println("Devo will not be able to watch for changes")
		return &Watcher{}
	}

	watcher := &Watcher{
		internalWatcher,
		make(chan fsnotify.Event),
	}
	go watcher.watch()
	return watcher
}

func (w *Watcher) Close() {
	if w.watcher == nil {
		return
	}
	err := w.watcher.Close()
	if err != nil {
		log.Println("Error closing watcher: ", err)
		return
	}
}

func (w *Watcher) Add(path string) {
	if w.watcher == nil {
		return
	}
	err := w.watcher.Add(path)
	if err != nil {
		log.Println("Error adding path to watcher: ", err)
	}
}

func (w *Watcher) watch() {
	if w.watcher == nil {
		return
	}

	var limiter = make(map[fsnotify.Event]*time.Timer)

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if timer, ok := limiter[event]; ok {
				timer.Reset(time.Second)
			} else {
				limiter[event] = time.AfterFunc(time.Second, func() {
					w.Events <- event
				})
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error while watching for changes:", err)
		}
	}
}
