package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/node"
	"github.com/knightfall22/matrose/scheduler"
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

	WorkerNodes []*node.Node
	Scheduler   scheduler.Scheduler

	lastWorker int
}

func New(workers []string, schedulerType string) *Manager {
	workerTaskMap := make(map[string][]uuid.UUID)

	var nodes []*node.Node
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}

		nAPI := fmt.Sprintf("http://%v", workers[worker])
		n := node.NewNode(workers[worker], nAPI, "worker")
		nodes = append(nodes, n)
	}

	var s scheduler.Scheduler
	switch schedulerType {
	case "roundrobin":
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	case "evpm":
		s = &scheduler.Epvm{Name: "evpm"}
	default:
		s = &scheduler.RoundRobin{Name: "roundrobin"}
	}

	return &Manager{
		Pending:       *queue.New(),
		TaskDb:        make(map[uuid.UUID]*task.Task),
		EventDb:       make(map[uuid.UUID]*task.TaskEvent),
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		Scheduler:     s,
		WorkerNodes:   nodes,
		TaskWorkerMap: make(map[uuid.UUID]string),
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	m.Pending.Enqueue(te)
}

// This method is responsible for looking at the requirements specified
// in a Task and evaluating the resources available in the pool of
// workers to see which worker is best suited to run the task.
func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)

	if candidates == nil {
		msg := fmt.Sprintf("No available candidates match resource request for task %v", t.ID)
		err := errors.New(msg)
		return nil, err
	}

	scores := m.Scheduler.Score(t, m.WorkerNodes)
	pick := m.Scheduler.Pick(scores, m.WorkerNodes)
	return pick, nil
}

func (m *Manager) updateTasks() {
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

			m.TaskDb[t.ID].HostPorts = t.HostPorts
			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking for task updates from workers")
		m.updateTasks()
		log.Println("Task updates completed")
		log.Println("Sleeping for 15 seconds")
		time.Sleep(15 * time.Second)
	}
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the queue")
		m.SendWork()
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}

func (m *Manager) stopTask(worker string, taskID string) {
	client := &http.Client{}
	url := fmt.Sprintf("http://%s/tasks/%s", worker, taskID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("error creating request to delete task %s: %v\n", taskID, err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error connecting to worker at %s: %v\n", url, err)
		return
	}

	if resp.StatusCode != 204 {
		log.Printf("Error sending request: %v\n", err)
		return
	}

	log.Printf("task %s has been scheduled to be stopped", taskID)
}

// Todo: add proper error handling
func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {

		event := m.Pending.Dequeue()

		taskEvent := event.(task.TaskEvent)

		sTask := taskEvent.Task

		//Add task and task event to appropriate stores
		m.EventDb[taskEvent.ID] = &taskEvent
		log.Printf("Pulled %v off pending queue", taskEvent)

		taskWorker, ok := m.TaskWorkerMap[sTask.ID]
		if ok {
			persistedTask := m.TaskDb[sTask.ID]
			if taskEvent.State == task.Completed &&
				task.ValidStateTransition(persistedTask.State, sTask.State) {
				m.stopTask(taskWorker, sTask.ID.String())
				return
			}

			log.Printf("invalid request: existing task %s is in state %v and cannot transition to the completed state\n",
				persistedTask.ID.String(), persistedTask.State,
			)
			return
		}

		sWorker, err := m.SelectWorker(sTask)
		if err != nil {
			log.Printf("error selecting worker for task %s: %v\n", sTask.ID, err)
		}

		m.WorkerTaskMap[sWorker.Name] = append(m.WorkerTaskMap[sWorker.Name], sTask.ID)
		m.TaskWorkerMap[sTask.ID] = sWorker.Name

		sTask.State = task.Scheduled
		m.TaskDb[sTask.ID] = &sTask

		data, err := json.Marshal(taskEvent)
		if err != nil {
			log.Printf("Unable to marshal task object: %+v.\n", sTask)
			return
		}

		url := fmt.Sprintf("http://%s/tasks", sWorker.Name)

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

		err = d.Decode(&t)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}

		log.Printf("%#v\n", t)
	} else {
		log.Println("Queue is empty")
	}
}

func (m *Manager) DoHealthChecks() {
	for {
		log.Println("Performing task health check")
		m.doHealthChecks()
		log.Println("Task health checks completed")
		log.Println("Sleeping for 60 seconds")
		time.Sleep(60 * time.Second)
	}
}

func (m *Manager) checkTaskHealth(t *task.Task) error {
	log.Printf("Calling health check for task %s: %s\n",
		t.ID, t.HealthCheck)

	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)
	if hostPort == nil {
		log.Printf("Have not collected task %s host port yet. Skipping.\n", t.ID)
		return nil
	}
	worker := strings.Split(w, ":")
	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)

	log.Printf("Calling health check for task %s: %s\n", t.ID, url)
	resp, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to health check %s", url)
		log.Println(msg)
		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Error health check for task %s did not return 200\n", t.ID)
		log.Println(msg)
		return errors.New(msg)
	}

	log.Printf("Task %s health check response: %v\n", t.ID, resp.StatusCode)
	return nil
}

func (m *Manager) doHealthChecks() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < 3 {
			err := m.checkTaskHealth(t)
			if err != nil {
				if t.RestartCount < 3 {
					m.restartTask(t)
				}
			}
		} else if t.State == task.Failed && t.RestartCount < 3 {
			m.restartTask(t)
		}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb[t.ID] = t

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}

	data, err := json.Marshal(te)
	if err != nil {
		log.Printf("Unable to marshal task object: %v.", t)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(t)
		return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		e := worker.ErrResponse{}
		err := d.Decode(&e)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}
		log.Printf("Response error (%d): %s", e.HTTPStatusCode, e.Message)
		return
	}

	newTask := task.Task{}
	err = d.Decode(&newTask)
	if err != nil {
		fmt.Printf("Error decoding response: %s\n", err.Error())
		return
	}

	log.Printf("%#v\n", t)
}

func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
		return &ports[k][0].HostPort
	}
	return nil
}

func (m *Manager) GetTasks() []*task.Task {
	taskLen := len(m.TaskDb)
	tasks := make([]*task.Task, taskLen)

	i := 0
	for _, task := range m.TaskDb {
		tasks[i] = task
		i++
	}

	return tasks
}
