package fsm

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type State struct {
	Name             string
	MsgTypeWhitelist string // example: "1, 2-10, 30"
	MsgTypeBlacklist string

	allowedMsgTypes map[uint32]bool
	transitions     map[uint32]*State
}

type StateTransition struct {
	FromState string
	ToState   string
	MsgType   uint32
}

type FiniteStateMachine struct {
	InitState   *string
	States      []State
	Transitions []StateTransition

	currentState *State
	stateNameMap map[string]*State
	lock         *sync.RWMutex
}

func parseMsgTypes(s string, f func(msgType uint32)) {
	if len(s) == 0 {
		return
	}

	for _, seg := range strings.Split(s, ",") {
		seg = strings.Trim(seg, " ")
		fromTo := strings.Split(seg, "-")
		if len(fromTo) == 1 {
			msgType, err := strconv.ParseUint(fromTo[0], 10, 32)
			if err != nil {
				logger.Errorf("Can't convert '%s' to uint32\n", fromTo[0])
			} else {
				f(uint32(msgType))
			}
		} else {
			fromType, err := strconv.ParseUint(fromTo[0], 10, 32)
			if err != nil {
				logger.Errorf("Can't convert '%s' to uint32\n", fromTo[0])
			} else {
				toType, err := strconv.ParseUint(fromTo[1], 10, 32)
				if err != nil {
					logger.Errorf("Can't convert '%s' to uint32\n", fromTo[1])
				} else {
					for i := fromType; i <= toType; i += 1 {
						f(uint32(i))
					}
				}

			}
		}
	}
}

var logger *zap.SugaredLogger

func Load(bytes []byte) (FiniteStateMachine, error) {
	if logger == nil {
		l, _ := zap.NewProduction()
		defer l.Sync()
		logger = l.Sugar()
	}

	var fsm FiniteStateMachine
	err := json.Unmarshal(bytes, &fsm)
	if err == nil {
		fsm.currentState = &fsm.States[0]
		fsm.stateNameMap = make(map[string]*State, len(fsm.States))
		fsm.lock = &sync.RWMutex{}

		for idx := range fsm.States {
			state := &fsm.States[idx]
			state.allowedMsgTypes = make(map[uint32]bool)
			state.transitions = make(map[uint32]*State)
			fsm.stateNameMap[state.Name] = state
			parseMsgTypes(state.MsgTypeWhitelist, func(msgType uint32) {
				state.allowedMsgTypes[msgType] = true
			})
			parseMsgTypes(state.MsgTypeBlacklist, func(msgType uint32) {
				state.allowedMsgTypes[msgType] = false
			})
		}

		for _, transition := range fsm.Transitions {
			fromState, exists := fsm.stateNameMap[transition.FromState]
			if !exists {
				logger.Errorf("invalid FromState in StateTransition: %s -> %s (%d)\n", transition.FromState, transition.ToState, transition.MsgType)
				continue
			}
			toState, exists := fsm.stateNameMap[transition.ToState]
			if !exists {
				logger.Errorf("invalid ToState in StateTransition: %s -> %s (%d)\n", transition.FromState, transition.ToState, transition.MsgType)
				continue
			}
			fromState.transitions[transition.MsgType] = toState
		}

		if fsm.InitState != nil {
			err = fsm.ChangeState(*fsm.InitState)
		}
	}
	return fsm, err
}

func (fsm *FiniteStateMachine) IsAllowed(msgType uint32) bool {
	fsm.lock.RLock()
	defer func() {
		fsm.lock.RUnlock()
	}()

	return fsm.currentState.allowedMsgTypes[msgType]
}

func (fsm *FiniteStateMachine) OnReceived(msgType uint32) {
	fsm.lock.Lock()
	defer func() {
		fsm.lock.Unlock()
	}()

	newState := fsm.currentState.transitions[msgType]
	if newState != nil {
		fsm.currentState = newState
	}
}

func (fsm *FiniteStateMachine) CurrentState() *State {
	fsm.lock.RLock()
	defer func() {
		fsm.lock.RUnlock()
	}()
	return fsm.currentState
}

func (fsm *FiniteStateMachine) ChangeState(name string) error {
	fsm.lock.Lock()
	defer func() {
		fsm.lock.Unlock()
	}()

	state, exists := fsm.stateNameMap[name]
	if exists {
		fsm.currentState = state
		return nil
	}
	return errors.New("Invalid state name: " + name)
}

func (fsm *FiniteStateMachine) MoveToNextState() bool {
	fsm.lock.Lock()
	defer func() {
		fsm.lock.Unlock()
	}()

	for i := 0; i < len(fsm.States)-1; i++ {
		if fsm.currentState == &fsm.States[i] {
			fsm.currentState = &fsm.States[i+1]
			return true
		}
	}
	return false
}
