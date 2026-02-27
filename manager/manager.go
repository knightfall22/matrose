package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/task"
	"github.com/knightfall22/matrose/worker"
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
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string

	lastWorker int
}

func New(workers []string) *Manager {
	workerTaskMap := make(map[string][]uuid.UUID)
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	fmt.Println("workers", workers)
	return &Manager{
		Pending:       *queue.New(),
		TaskDb:        make(map[uuid.UUID]*task.Task),
		EventDb:       make(map[uuid.UUID]*task.TaskEvent),
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: make(map[uuid.UUID]string),
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

// This method is responsible for looking at the requirements specified
// in a Task and evaluating the resources available in the pool of
// workers to see which worker is best suited to run the task.
func (m *Manager) SelectWorker() string {
	newWorker := 0
	if m.lastWorker+1 < len(m.Workers) {
		newWorker = m.lastWorker + 1
		m.lastWorker++
	} else {
		m.lastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) UpdateTasks() {
	for _, worker := range m.Workers {
		log.Printf("Checking worker %v for task updates", worker)

		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error connecting to %v: %v\n", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error sending request: %v\n", err)
		}

		d := json.NewDecoder(resp.Body)
		var taskList []*task.Task
		err = d.Decode(&taskList)
		if err != nil {
			log.Printf("Error unmarshalling tasks: %s\n", err.Error())
		}

		for _, t := range taskList {
			log.Printf("Attempting to update task %v\n", t.ID)

			_, ok := m.TaskDb[t.ID]
			if !ok {
				log.Printf("Task with ID %s not found\n", t.ID)
				return
			}

			if m.TaskDb[t.ID].State != t.State {
				m.TaskDb[t.ID].State = t.State
			}

			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}
}

// Todo: add proper error handling
func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		sWorker := m.SelectWorker()

		event := m.Pending.Dequeue()

		taskEvent := event.(task.TaskEvent)
		sTask := taskEvent.Task

		//Add task and task event to appropriate stores
		m.EventDb[taskEvent.ID] = &taskEvent
		m.WorkerTaskMap[sWorker] = append(m.WorkerTaskMap[sWorker], sTask.ID)
		m.TaskWorkerMap[sTask.ID] = sWorker

		sTask.State = task.Scheduled
		m.TaskDb[sTask.ID] = &sTask

		data, err := json.Marshal(taskEvent)
		if err != nil {
			log.Printf("Unable to marshal task object: %+v.\n", sTask)
			return
		}

		url := fmt.Sprintf("http://%s/tasks", sWorker)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Error connecting to %v: %v\n", sWorker, err)
			m.Pending.Enqueue(taskEvent)
			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := d.Decode(&e)
			if err != nil {
				fmt.Printf("Error decoding response: %s\n", err.Error())
				return
			}

			log.Printf("Response error (%d): %s", e.HTTPStatusCode, e.Message)
			return
		}

		t := task.Task{}

		err = d.Decode(t)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("%#v\n", t)
	} else {
		log.Println("Queue is empty")
	}
}
