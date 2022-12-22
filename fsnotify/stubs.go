package fsnotify

type Op uint32

type Event struct {
	Name string
	Op   Op
}

const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

type Watcher struct {
	Events chan Event
	Errors chan error
}

func (Watcher) Add(string) {
	panic("fsnotify is not supported")
}

func (Watcher) Close() {
	panic("fsnotify is not supported")
}

func NewWatcher() (*Watcher, error) {
	panic("fsnotify is not supported")
}
