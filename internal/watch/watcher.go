package watch

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/bep/debounce"
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
	Ignore(fn func(path string) bool)
}

func New(log *slog.Logger, dir string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &FileWatcher{
		log:           log,
		watcher:       watcher,
		ignores:       []func(path string) bool{},
		duplicates:    map[string]struct{}{},
		pendingEvents: []Event{},
		debounce:      debounce.New(debounceDelay),
		errorCh:       make(chan error),
	}
	if err := fw.addPath(dir); err != nil {
		return nil, err
	}
	return fw, nil
}

var _ Watcher = (*FileWatcher)(nil)

type FileWatcher struct {
	log           *slog.Logger
	watcher       *fsnotify.Watcher
	duplicates    map[string]struct{}
	ignores       []func(path string) bool
	pendingEvents []Event
	debounce      func(func())
	errorCh       chan error
}

func (w *FileWatcher) Close() error {
	return w.watcher.Close()
}

func (w *FileWatcher) Ignore(fn func(path string) bool) {
	w.ignores = append(w.ignores, fn)
}

func (w *FileWatcher) Watch(fn func(events []Event) error) error {
	for {
		select {
		case event := <-w.watcher.Events:
			if w.shouldIgnore(event.Name) {
				continue
			}
			w.pendingEvents = append(w.pendingEvents, event)
			w.debounce(func() {
				err := w.handleEvents(w.pendingEvents, fn)
				w.pendingEvents = nil
				if err != nil {
					w.errorCh <- err
				}
			})
		case err := <-w.errorCh:
			if err := w.handleError(err); err != nil {
				return err
			}
		case err := <-w.watcher.Errors:
			// Watcher was closed
			if err == nil {
				return nil
			}
			if err := w.handleError(err); err != nil {
				return err
			}
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

func (w *FileWatcher) handleError(err error) error {
	w.log.Warn("watch error", slog.String("err", err.Error()))
	return nil
}

func (w *FileWatcher) shouldIgnore(path string) bool {
	for _, ignore := range w.ignores {
		if ignore(path) {
			return true
		}
	}
	return false
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
		if !d.IsDir() {
			continue
		}
		eg.Go(func() error {
			return w.addDir(filepath.Join(dir, d.Name()))
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
