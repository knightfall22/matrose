package worker

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/task"
)

// A worker is responsible for running the tasks assigned
// to it by the manager. If a task fails for
// any reason, it must attempt to restart the task. The worker also
// makes metrics about its tasks and overall machine health available
// for the manager to poll.
// Workers:
//   - Run tasks as Docker containers
//   - Accept tasks to run from a manager
//   - Provide relevant statistics to the manager for the purpose of scheduling tasks
//   - Keep track of its tasks and their state
type Worker struct {
	Name  string
	Queue queue.Queue

	//Keeps tracks of all tracks in a worker
	Db map[uuid.UUID]*task.Task

	TaskCount int
}

func (w *Worker) CollectStats() {
	fmt.Println("I will collect stats")
}

// Responsible for run task. Keeps track of the task state and responds in kind.
func (w *Worker) RunTask() {
	fmt.Println("I will run a task")
}

func (w *Worker) StartTask() {
	fmt.Println("I will start a task")
}

func (w *Worker) StopTask() {
	fmt.Println("I will stop a task")
}
