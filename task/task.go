package task

import (
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

//Task is the smallest unit of work in an orchestration system and typically runs a container.
//The container are ran on Docker
//A task should specify the following:
// - The amount of memory, CPU, and disk it needs to run effectively
// - What the orchestrator should do in case of failures, typically called a restart policy
// - The name of the container image used to run the task

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID    uuid.UUID
	Name  string
	State State
	Image string

	// Memory and Disk will help the system identify the number of resources a task needs.
	Memory int
	Disk   int

	// `ExposedPorts` and `PortBindings` are used by
	// Docker to ensure the machine allocates the proper network ports for
	// the task and that it is available on the network.
	ExposedPorts nat.PortSet
	PortBindings map[string]string

	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
}

// Serves a state transition mechanism.
type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}
