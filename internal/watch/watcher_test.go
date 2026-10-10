package watch_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/matthewmueller/jig/internal/watch"
	"github.com/matthewmueller/logs"
)

func TestWatch(t *testing.T) {
	is := is.New(t)
	dir := t.TempDir()
	watcher, err := watch.New(logs.Default(), dir, func(path string) bool {
		return filepath.Base(path) == "ignored.txt"
	})
	is.NoErr(err)
	events := make(chan []watch.Event, 100)
	done := make(chan error, 1)
	go func() {
		done <- watcher.Watch(func(batch []watch.Event) error {
			events <- batch
			return nil
		})
	}()
	// Write around the debounce delay so batches flush while events arrive
	for i := range 30 {
		is.NoErr(os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte(strconv.Itoa(i)), 0644))
		is.NoErr(os.WriteFile(filepath.Join(dir, "a.txt"), []byte(strconv.Itoa(i)), 0644))
		time.Sleep(time.Duration(i%4) * 10 * time.Millisecond)
	}
	select {
	case batch := <-events:
		is.True(len(batch) > 0)
		for _, event := range batch {
			is.Equal(filepath.Base(event.Name), "a.txt")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for events")
	}
	is.NoErr(watcher.Close())
	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(5 * time.Second):
		t.Fatal("watch didn't return after close")
	}
}
