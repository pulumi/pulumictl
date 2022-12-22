package notify

type Event uint32

type EventInfo interface {
	Event() Event
	Path() string
	Sys() interface{}
}

func Stop(c chan<- EventInfo) {
	panic("notify is not supported")
}

const All = 0

func Watch(path string, c chan<- EventInfo, events ...Event) error {
	panic("notify is not supported")
}
