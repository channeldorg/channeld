package channeld

import (
	"channeld.clewcat.com/channeld/pkg/common"
)

var Event_GlobalChannelPossessed = &Event[*Channel]{}
var Event_GlobalChannelUnpossessed = &Event[struct{}]{}
var Event_ChannelCreated = &Event[*Channel]{}
var Event_ChannelRemoved = &Event[common.ChannelId]{}

type EventData interface {
}

type Event[T EventData] struct {
	handlers []func(data T)
}

func (e *Event[T]) Listen(handler func(data T)) {
	if e.handlers == nil {
		e.handlers = make([]func(data T), 0)
	}
	e.handlers = append(e.handlers, handler)
}

func (e *Event[T]) Broadcast(data T) {
	for _, handler := range e.handlers {
		handler(data)
	}
}
