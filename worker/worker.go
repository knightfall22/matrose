package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/knightfall22/matrose/stats"
	"github.com/knightfall22/matrose/store"
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
	Db store.Store

	Stats *stats.Stats

	TaskCount int
}

func New(name, dbType string) (*Worker, error) {
	w := &Worker{
		Name:  name,
		Queue: *queue.New(),
	}

	var s store.Store
	var err error
	switch dbType {
	case "memory":
		s = store.NewInMemoryTaskStore()
	case "persisted":
		filename := fmt.Sprintf("%s_tasks.db", name)
		s, err = store.NewTaskStore(filename, 0600, "tasks")
		if err != nil {
			log.Printf("error starting up persisted task worker for %s, %v\n", w.Name, err)
			return nil, err
		}
	default:
		s = store.NewInMemoryTaskStore()
	}

	w.Db = s
	return w, nil
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) CollectStats() {
	for {
		log.Println("Collecting Stats")
		w.Stats = stats.GetStats()
		w.Stats.TaskCount = w.TaskCount
		time.Sleep(15 * time.Second)
	}
}

// Responsible for run task. Keeps track of the task state and responds in kind.
func (w *Worker) runTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println("No tasks in the queue")
		return task.DockerResult{
			Error:  nil,
			Result: "queue empty",
		}
	}

	taskQueued := t.(task.Task)
	fmt.Printf("Found task in queue: %v:\n", taskQueued)

	err := w.Db.Put(taskQueued.ID.String(), &taskQueued)
	if err != nil {
		msg := fmt.Errorf("error storing task %s: %v",
			taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	queuedTask, err := w.Db.Get(taskQueued.ID.String())
	if err != nil {
		msg := fmt.Errorf("error getting task %s from database: %v",
			taskQueued.ID.String(), err)
		log.Println(msg)
		return task.DockerResult{Error: msg}
	}

	taskPersisted := *queuedTask.(*task.Task)

	if taskPersisted.State == task.Completed {
		return w.StopTask(taskPersisted)
	}

	var result task.DockerResult
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			if taskPersisted.ContainerID != "" {
				result := w.StopTask(taskPersisted)
				if result.Error != nil {
					log.Printf("%v\n", result.Error)
				}
			}
			result = w.StartTask(taskQueued)
		default:
			result.Error = errors.New("We should not get here")
		}
	} else {
		err := fmt.Errorf("Invalid transition from %v to %v",
			taskPersisted.State, taskQueued.State)
		result.Error = err
	}

	return result
}

func (w *Worker) RunTask() task.DockerResult {
	for {
		if w.Queue.Len() != 0 {
			res := w.runTask()
			if res.Error != nil {
				log.Printf("Error running task: %v\n", res.Error)
			}
		} else {
			log.Printf("No tasks to process currently.\n")
		}

		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	res := d.Run()
	if res.Error != nil {
		log.Printf("Error stopping container %v: %v\n", t.ContainerID, res.Error)
		t.State = task.Failed
		w.Db.Put(t.ID.String(), &t)
		return res
	}

	t.ContainerID = res.ContainerID
	t.State = task.Running
	t.FinishTime = time.Now().UTC()
	w.Db.Put(t.ID.String(), &t)
	return res
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)

	res := d.Stop(t.ContainerID)
	if res.Error != nil {
		log.Printf("Error stopping container %v: %v\n", t.ContainerID, res.Error)
		return res
	}

	t.State = task.Completed
	t.FinishTime = time.Now().UTC()
	w.Db.Put(t.ID.String(), &t)

	log.Printf("Stopped and removed container %v for task %v\n", t.ContainerID, t.ID)
	return res
}

func (w *Worker) GetTasks() []*task.Task {
	tasks, err := w.Db.List()
	if err != nil {
		log.Printf("error getting list of tasks: %v\n", err)
		return nil
	}

	return tasks.([]*task.Task)
}

func (w *Worker) InspectTask(t *task.Task) task.DockerInspectResponse {
	config := task.NewConfig(t)
	d := task.NewDocker(config)

	return d.Inspect(t.ContainerID)
}

func (w *Worker) updateTasks() {
	tasks, err := w.Db.List()
	if err != nil {
		log.Printf("error getting list of tasks: %v\n", err)
		return
	}

	for id, t := range tasks.([]*task.Task) {
		if t.State == task.Running {
			resp := w.InspectTask(t)
			if resp.Error != nil {
				fmt.Printf("ERROR: %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf("No container for running task %s\n", id)
				t.State = task.Failed
			}

			if resp.Container.State.Status == "exited" {
				log.Printf("Container for task %s in non-running state %s",
					id, resp.Container.State.Status)
				t.State = task.Failed
			}

			t.HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports

			w.Db.Put(t.ID.String(), t)
		}
	}
}

func (w *Worker) UpdateTasks() {
	for {
		log.Println("Checking status of tasks")
		w.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}
