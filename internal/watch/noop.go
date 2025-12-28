package watch

func Noop() Watcher {
	return noop{}
}

type noop struct {
}

var _ Watcher = (*noop)(nil)

func (noop) Close() error {
	return nil
}

func (noop) Ignore(fn func(path string) bool) {
}

func (noop) Watch(fn func(events []Event) error) error {
	return nil
}
