package main

import (
	"fmt"
	"os"
	"time"

	"github.com/knightfall22/matrose/task"
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
	dockerTask, createResult := createContainer()
	if createResult.Error != nil {
		fmt.Printf("%v", createResult.Error)
		os.Exit(1)
	}

	time.Sleep(time.Second * 5)
	fmt.Printf("stopping container %s\n", createResult.ContainerID)
	_ = containerStop(dockerTask, createResult.ContainerID)
}
