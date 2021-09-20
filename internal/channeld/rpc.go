package channeld

import "sync"

type StubId = uint32

type RPCService struct {
	stubs map[StubId]*Connection
}

var instance *RPCService
var once sync.Once

func RPC() *RPCService {
	once.Do(func() {
		instance = &RPCService{make(map[StubId]*Connection)}
	})
	return instance
}

func (s *RPCService) SaveStub(stubId StubId, c *Connection) {
	s.stubs[stubId] = c
}

func (s *RPCService) Send(c *Connection, msgType uint32, m Message, callback func(Message)) {

}
