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

	w1 := &worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wapi1 := worker.Api{
		Address: whost,
		Port:    wport,
		Worker:  w1,
	}

	w2 := &worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wapi2 := worker.Api{
		Address: whost,
		Port:    mport + 1,
		Worker:  w2,
	}

	w3 := &worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	wapi3 := worker.Api{
		Address: whost,
		Port:    mport + 2,
		Worker:  w3,
	}

	go w1.RunTask()
	go w1.CollectStats()
	go w1.UpdateTasks()
	go wapi1.StartServer()
	time.Sleep(5 * time.Second)

	go w2.RunTask()
	go w2.CollectStats()
	go w2.UpdateTasks()
	go wapi2.StartServer()
	time.Sleep(5 * time.Second)

	go w3.RunTask()
	go w3.CollectStats()
	go w3.UpdateTasks()
	go wapi3.StartServer()
	time.Sleep(5 * time.Second)

	workers := []string{
		fmt.Sprintf("%s:%d", whost, wport),
		fmt.Sprintf("%s:%d", whost, mport+1),
		fmt.Sprintf("%s:%d", whost, mport+2),
	}
	m := manager.New(workers, "evpm")
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
