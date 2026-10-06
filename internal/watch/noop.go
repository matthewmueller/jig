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

func (noop) Watch(fn func(events []Event) error) error {
	return nil
}
