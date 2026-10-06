package watch

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sync/errgroup"
)

// File event
type Event = fsnotify.Event

// Ops
var (
	Remove = fsnotify.Remove
	Write  = fsnotify.Write
	Create = fsnotify.Create
)

// Arbitrarily picked after some manual testing. OSX is pretty fast, but Ubuntu
// requires a longer delay for writes. Duplicate checks below allow us to keep
// this snappy.
var debounceDelay = 20 * time.Millisecond

type Watcher interface {
	Close() error
	Watch(fn func(events []Event) error) error
}

// New watches dir recursively, skipping ignored paths
func New(log *slog.Logger, dir string, ignore func(path string) bool) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FileWatcher{
		log:     log,
		watcher: watcher,
		ignore:  ignore,
	}
	if err := fw.addPath(dir); err != nil {
		return nil, err
	}
	return fw, nil
}

var _ Watcher = (*FileWatcher)(nil)

type FileWatcher struct {
	log     *slog.Logger
	watcher *fsnotify.Watcher
	ignore  func(path string) bool
}

func (w *FileWatcher) Close() error {
	return w.watcher.Close()
}

// Watch calls fn with batches of events until the watcher is closed. Events
// are debounced and handled in this goroutine, so fn calls never overlap.
func (w *FileWatcher) Watch(fn func(events []Event) error) error {
	var pending []Event
	timer := time.NewTimer(debounceDelay)
	timer.Stop()
	defer timer.Stop()
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if w.ignore(event.Name) {
				continue
			}
			pending = append(pending, event)
			timer.Reset(debounceDelay)
		case <-timer.C:
			if err := w.handleEvents(pending, fn); err != nil {
				w.log.Warn("watch error", slog.String("err", err.Error()))
			}
			pending = nil
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			w.log.Warn("watch error", slog.String("err", err.Error()))
		}
	}
}

func (w *FileWatcher) handleEvents(events []Event, fn func(events []Event) error) error {
	if len(events) == 0 {
		return nil
	}
	uniqueEvents := make([]Event, 0, len(events))
	seen := map[string]struct{}{}
	for _, event := range slices.Backward(events) {
		if _, ok := seen[event.Name]; ok {
			continue
		}
		seen[event.Name] = struct{}{}
		uniqueEvents = append(uniqueEvents, event)
	}
	if err := fn(uniqueEvents); err != nil {
		return err
	}
	// Handle create events
	for _, event := range uniqueEvents {
		if event.Op&fsnotify.Create == fsnotify.Create {
			if err := w.addPath(event.Name); err != nil {
				return err
			}
		}
	}
	// Handle delete events
	for _, event := range uniqueEvents {
		if event.Op&fsnotify.Remove == fsnotify.Remove {
			if err := w.removeDir(event.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *FileWatcher) addPath(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !stat.IsDir() {
		return w.watcher.Add(path)
	}
	return w.addDir(path)
}

func (w *FileWatcher) addDir(dir string) error {
	if err := w.watcher.Add(dir); err != nil {
		return err
	}
	des, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	eg := new(errgroup.Group)
	for _, d := range des {
		path := filepath.Join(dir, d.Name())
		if !d.IsDir() || w.ignore(path) {
			continue
		}
		eg.Go(func() error {
			return w.addDir(path)
		})
	}
	return eg.Wait()
}

func (w *FileWatcher) removeDir(dir string) error {
	// Don't try removing a file that doesn't exist
	// if _, err := os.Stat(dir); err != nil {
	// 	if os.IsNotExist(err) {
	// 		return nil
	// 	}
	// }
	if err := w.watcher.Remove(dir); err != nil {
		// if errors.Is(err, fsnotify.ErrNonExistentWatch) {
		// 	return nil
		// }
		return err
	}
	return nil
}
