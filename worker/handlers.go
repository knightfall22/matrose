package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/knightfall22/matrose/task"
)

func (a *Api) initRouter() {
	a.Router = http.NewServeMux()

	a.Router.HandleFunc("POST /tasks", a.StartTaskHandler)
	a.Router.HandleFunc("GET /tasks", a.GetTasksHandler)
	a.Router.HandleFunc("DELETE /tasks/{id}", a.StopTaskHandler)
	a.Router.HandleFunc("GET /stats", a.GetStatsHandler)
}

func (a *Api) StartServer() {
	a.initRouter()

	log.Printf("server started on: %s:%d\n", a.Address, a.Port)
	if err := http.ListenAndServe(
		fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router,
	); err != nil {
		log.Fatalf("server failed: %v\n", err)
	}
}

func (a *Api) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf(msg)
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %v\n", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *Api) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *Api) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the ID from the URL path
	idStr := r.PathValue("id")

	if idStr == "" {
		log.Printf("No taskID passed in request.\n")
		w.WriteHeader(http.StatusBadRequest)
	}

	taskId, err := uuid.Parse(idStr)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v\n", err)
		log.Printf(msg)
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	res, err := a.Worker.Db.Get(taskId.String())
	if err != nil {
		msg := fmt.Sprintf("No task with ID %v found", taskId)
		log.Printf(msg)
		writeError(w, http.StatusNotFound, msg)
		return
	}

	taskToStop, ok := res.(*task.Task)
	if !ok {
		log.Printf("cannot convert result %v to task.Task type\n", taskToStop)
		return
	}
	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	taskCopy.FinishTime = time.Now().UTC()

	a.Worker.AddTask(taskCopy)
	log.Printf("Added task %v to stop container %v\n", taskToStop.ID, taskToStop.ContainerID)
	w.WriteHeader(http.StatusNoContent)
	return
}

func (a *Api) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}

type ErrResponse struct {
	HTTPStatusCode int
	Message        string
}

func writeError(w http.ResponseWriter, code int, message string) {
	log.Printf(message)
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrResponse{
		HTTPStatusCode: code,
		Message:        message,
	})
}
