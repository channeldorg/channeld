package channeld

import (
	"sync"

	"github.com/metaworking/channeld/pkg/channeldpb"
	"github.com/metaworking/channeld/pkg/common"
)

var Event_GlobalChannelPossessed = &Event[*Channel]{}
var Event_GlobalChannelUnpossessed = &Event[struct{}]{}
var Event_ChannelCreated = &Event[*Channel]{}
var Event_ChannelRemoving = &Event[*Channel]{}
var Event_ChannelRemoved = &Event[common.ChannelId]{}

type AuthEventData struct {
	AuthResult            channeldpb.AuthResultMessage_AuthResult
	Connection            ConnectionInChannel
	PlayerIdentifierToken string
}

var Event_AuthComplete = &Event[AuthEventData]{}

var Event_FsmDisallowed = &Event[*Connection]{}

type EntityChannelSpatiallyOwnedEventData struct {
	EntityChannel *Channel
	SpatialChanel *Channel
}

var Event_EntityChannelSpatiallyOwned = &Event[EntityChannelSpatiallyOwnedEventData]{}

type EventData interface {
}

type eventHandler[T EventData] struct {
	owner       interface{}
	handlerFunc func(data T)
	triggerOnce bool
}

type Event[T EventData] struct {
	handlersLock sync.RWMutex
	handlers     []*eventHandler[T]
}

func (e *Event[T]) Listen(handlerFunc func(data T)) {
	e.ListenFor(nil, handlerFunc)
}

func (e *Event[T]) ListenOnce(handlerFunc func(data T)) {
	e.handlersLock.Lock()
	defer e.handlersLock.Unlock()
	if e.handlers == nil {
		e.handlers = make([]*eventHandler[T], 0)
	}
	e.handlers = append(e.handlers, &eventHandler[T]{nil, handlerFunc, true})
}

func (e *Event[T]) ListenFor(owner interface{}, handlerFunc func(data T)) {
	e.handlersLock.Lock()
	defer e.handlersLock.Unlock()
	if e.handlers == nil {
		e.handlers = make([]*eventHandler[T], 0)
	}
	e.handlers = append(e.handlers, &eventHandler[T]{owner, handlerFunc, false})
}

func (e *Event[T]) UnlistenFor(owner interface{}) {
	e.handlersLock.Lock()
	defer e.handlersLock.Unlock()
	for i, handler := range e.handlers {
		if handler.owner == owner {
			e.handlers = append(e.handlers[:i], e.handlers[i+1:]...)
		}
	}
}

func (e *Event[T]) Wait() chan T {
	ch := make(chan T)
	e.ListenOnce(func(data T) {
		ch <- data
	})
	return ch
}

func (e *Event[T]) Broadcast(data T) {
	e.handlersLock.RLock()
	defer e.handlersLock.RUnlock()
	for i, handler := range e.handlers {
		handler.handlerFunc(data)
		if handler.triggerOnce {
			e.handlers = append(e.handlers[:i], e.handlers[i+1:]...)
		}
	}
}
