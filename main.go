package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/manager"
	"github.com/knightfall22/matrose/task"
	"github.com/knightfall22/matrose/worker"
)

func main() {
	whost := os.Getenv("CUBE_WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("CUBE_WORKER_PORT"))

	mhost := os.Getenv("CUBE_MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("CUBE_MANAGER_PORT"))

	fmt.Println("Starting Cube worker")

	w := &worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wapi := worker.Api{
		Address: whost,
		Port:    wport,
		Worker:  w,
	}

	go w.RunTask()
	go w.CollectStats()
	go w.UpdateTasks()
	go wapi.StartServer()
	time.Sleep(5 * time.Second)

	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)
	mapi := manager.Api{
		Address: mhost,
		Port:    mport,
		Manager: m,
	}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.StartServer()
}
