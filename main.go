package main

import (
	"fmt"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/knightfall22/matrose/task"
	"github.com/knightfall22/matrose/worker"
	"github.com/moby/moby/client"
)

func createContainer() (*task.Docker, *task.DockerResult) {
	c := task.Config{
		Name:  "test-container-1",
		Image: "postgres:13",
		ENV: []string{
			"POSTGRES_USER=matty",
			"POSTGRES_PASSWORD=sailor",
		},
	}

	dc, _ := client.NewClientWithOpts(client.FromEnv)
	d := task.Docker{
		Client: dc,
		Config: c,
	}

	result := d.Run()
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil, nil
	}

	fmt.Printf(
		"Container %s is running with config %v\n", result.ContainerID, c)
	return &d, &result

}

func containerStop(d *task.Docker, id string) *task.DockerResult {
	res := d.Stop(id)
	if res.Error != nil {
		fmt.Printf("%v\n", res.Error)
		return nil
	}

	fmt.Printf(
		"Container %s has been stopped and removed\n", res.ContainerID)
	return &res
}
func main() {
	fmt.Printf("create a test container\n")
	// 	dockerTask, createResult := createContainer()
	// 	if createResult.Error != nil {
	// 		fmt.Printf("%v", createResult.Error)
	// 		os.Exit(1)
	// 	}

	//		time.Sleep(time.Second * 5)
	//		fmt.Printf("stopping container %s\n", createResult.ContainerID)
	//		_ = containerStop(dockerTask, createResult.ContainerID)
	//	}
	db := make(map[uuid.UUID]*task.Task)
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    db,
	}

	t := task.Task{
		ID:    uuid.New(),
		Name:  "test-container-1",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}

	// first time the worker will see the task
	fmt.Println("starting task")
	w.AddTask(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}

	t.ContainerID = result.ContainerID
	fmt.Printf("task %s is running in container %s\n", t.ID, t.ContainerID)
	fmt.Println("Sleepy time")
	time.Sleep(time.Second * 30)

	fmt.Printf("stopping task %s\n", t.ID)
	t.State = task.Completed
	w.AddTask(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}
}
