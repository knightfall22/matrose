package manager

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/task"
)

// To run jobs in the orchestration system, users submit
// their jobs to the manager. The manager, using the scheduler, then
// finds a machine where the job’s tasks can run. The manager also
// periodically collects metrics from each of its workers, which are used
// in the scheduling process.
// The manager does the following:
//   - Accept requests from users to start and stop tasks.
//   - Schedule tasks onto worker machines.
//   - Keep track of tasks, their states, and the machine on which they run.
type Manager struct {
	Pending       queue.Queue
	TaskDb        map[string][]*task.Task
	EventDb       map[string][]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

// This method is responsible for looking at the requirements specified
// in a Task and evaluating the resources available in the pool of
// workers to see which worker is best suited to run the task.
func (m *Manager) SelectWorker() {
	fmt.Println("I will select an appropriate worker")
}

func (m *Manager) UpdateTasks() {
	fmt.Println("I will update tasks")
}

func (m *Manager) SendWork() {
	fmt.Println("I will send work to workers")
}
