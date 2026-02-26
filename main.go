package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/task"
	"github.com/knightfall22/matrose/worker"
)

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			res := w.RunTask()
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

func main() {
	host := os.Getenv("CUBE_HOST")
	port, _ := strconv.Atoi(os.Getenv("CUBE_PORT"))

	fmt.Println("Starting Cube worker")

	w := &worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	api := worker.Api{
		Address: host,
		Port:    port,
		Worker:  w,
	}

	go runTasks(w)
	go w.CollectStats()
	api.StartServer()

}
