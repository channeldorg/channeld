package channeld

import (
	"sync"

	"channeld.clewcat.com/channeld/pkg/common"
)

var Event_GlobalChannelPossessed = &Event[*Channel]{}
var Event_GlobalChannelUnpossessed = &Event[struct{}]{}
var Event_ChannelCreated = &Event[*Channel]{}
var Event_ChannelRemoved = &Event[common.ChannelId]{}

type EventData interface {
}

type Event[T EventData] struct {
	handlersLock sync.RWMutex
	handlers     []func(data T)
}

func (e *Event[T]) Listen(handler func(data T)) {
	e.handlersLock.Lock()
	defer e.handlersLock.Unlock()
	if e.handlers == nil {
		e.handlers = make([]func(data T), 0)
	}
	e.handlers = append(e.handlers, handler)
}

func (e *Event[T]) Wait() chan T {
	ch := make(chan T)
	e.Listen(func(data T) {
		ch <- data
	})
	return ch
}

func (e *Event[T]) Broadcast(data T) {
	e.handlersLock.RLock()
	defer e.handlersLock.RUnlock()
	for _, handler := range e.handlers {
		handler(data)
	}
}
