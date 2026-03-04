package task

import (
	"fmt"
	"slices"
)

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

func (s State) String() string {
	states := []string{"Pending", "Scheduled", "Running", "Completed", "Failed"}

	if s < Pending || int(s) >= len(states) {
		return fmt.Sprintf("unknown state(%d)", s)
	}

	return states[s]
}

var stateTransitionMap = map[State][]State{
	Pending:   []State{Scheduled},
	Scheduled: []State{Scheduled, Running, Failed},
	Running:   []State{Running, Completed, Failed, Scheduled},
	Completed: []State{},
	Failed:    []State{Scheduled},
}

func Contains(states []State, state State) bool {
	return slices.Contains(states, state)
}

func ValidStateTransition(src State, dst State) bool {
	return Contains(stateTransitionMap[src], dst)
}
