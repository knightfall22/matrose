package task

import (
	"context"
	"io"
	"log"
	"math"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/moby/moby/pkg/stdcopy"
)

//Task is the smallest unit of work in an orchestration system and typically runs a container.
//The container are ran on Docker
//A task should specify the following:
// - The amount of memory, CPU, and disk it needs to run effectively
// - What the orchestrator should do in case of failures, typically called a restart policy
// - The name of the container image used to run the task

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	State State     `json:"state"`
	Image string    `json:"image"`

	ContainerID string `json:"container_id"`

	CPU float64 `json:"cpu"`

	// Memory and Disk will help the system identify the number of resources a task needs.
	Memory int64 `json:"memory"`
	Disk   int64 `json:"disk"`

	// `ExposedPorts` and `PortBindings` are used by
	// Docker to ensure the machine allocates the proper network ports for
	// the task and that it is available on the network.
	ExposedPorts nat.PortSet       `json:"exposed_ports"`
	PortBindings map[string]string `json:"port_bindings"`

	HostPorts nat.PortMap `json:"host_ports"`

	HealthCheck  string `json:"health_check"`
	RestartCount int

	RestartPolicy string    `json:"restart_policy"`
	StartTime     time.Time `json:"start_time"`
	FinishTime    time.Time `json:"finish_time"`
}

// Serves a state transition mechanism.
type TaskEvent struct {
	ID        uuid.UUID `json:"id"`
	State     State     `json:"state"`
	Timestamp time.Time `json:"timestamp"`
	Task      Task      `json:"task"`
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		CPU:           t.CPU,
		Memory:        t.Memory,
		Disk:          t.Disk,
		ExposedPorts:  t.ExposedPorts,
		PortBindings:  t.PortBindings,
		Image:         t.Image,
		RestartPolicy: t.RestartPolicy,
	}
}

// Contains information about the configuration of the task
type Config struct {
	// Name of the task. Used to identify the task,
	// also the name of the running container
	Name         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool

	ExposedPorts nat.PortSet
	PortBindings map[string]string

	Cmd []string

	//Holds the name of the image to run
	Image string

	//The scheduler will use `Memory`, `Disk`, `CPU` to find a node in
	// the cluster capable of running a task.
	// They will also be used to tell the Docker daemon the number of
	// resources a task requires.
	CPU    float64
	Memory int64
	Disk   int64

	// Allows a user to specify environment variables that
	// will get passed into the container.
	ENV []string

	// RestartPolicy field tells the Docker daemon what to
	// do if a container dies unexpectedly.Can be either "",
	// `always`, `unless-stopped`, or `on-failure`. Setting this
	// field to always will, as its name implies, restart a container if it
	// stops. Setting it to unless-stopped will restart a container
	// unless it has been stopped (e.g., by docker stop). Setting it to
	// on-failure will restart the container if it exits due to an error
	// (i.e., a nonzero exit code).
	RestartPolicy string
}

type DockerResult struct {
	Error error
	//TODO: Convert action into an enum
	Action      string
	ContainerID string
	Result      string
}

type DockerInspectResponse struct {
	Error     error
	Container *container.InspectResponse
}

// Encapsulates everything we need to run our task as a Docker container.
type Docker struct {
	Client *client.Client
	Config Config
}

// Connects with docker from the environment
func NewDocker(c *Config) *Docker {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	return &Docker{
		Client: dc,
		Config: *c,
	}
}

func (d *Docker) Inspect(containerID string) DockerInspectResponse {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	ctx := context.Background()
	resp, err := dc.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("Error inspecting container: %s\n", err)
		return DockerInspectResponse{Error: err}
	}

	return DockerInspectResponse{Container: &resp}
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(
		ctx, d.Config.Image, image.PullOptions{},
	)
	if err != nil {
		log.Printf("Error pulling image %s: %v\n", err)
		return DockerResult{Error: err}
	}

	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}

	r := container.Resources{
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.CPU * math.Pow(10, 9)),
	}

	cc := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.ENV,
		ExposedPorts: d.Config.ExposedPorts,
	}

	portBindings := make(nat.PortMap)
	for k, v := range d.Config.PortBindings {
		portBindings[nat.Port(k)] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0", // Bind to all host interfaces
				HostPort: v,
			},
		}
	}

	hc := container.HostConfig{
		RestartPolicy: rp,
		Resources:     r,
		PortBindings:  portBindings,
	}
	resp, err := d.Client.ContainerCreate(
		ctx,
		&cc,
		&hc,
		nil,
		nil,
		d.Config.Name,
	)
	if err != nil {
		log.Printf("Error creating container using image %s: %v\n", d.Config.Image, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		log.Printf("Error starting container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	// d.Config.Runtime.ContainerID = resp.ID

	out, err := d.Client.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		log.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
		return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return DockerResult{
		ContainerID: resp.ID,
		Action:      "start",
		Result:      "success",
	}
}

func (d *Docker) Stop(id string) DockerResult {
	log.Printf("Attempting to stop container %v", id)
	ctx := context.Background()

	err := d.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		log.Printf("Error stopping container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	err = d.Client.ContainerRemove(ctx, id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         false,
	})
	if err != nil {
		log.Printf("Error removing container %s: %v\n", id, err)
		return DockerResult{Error: err}
	}

	return DockerResult{Action: "stop", Error: nil, Result: "success"}
}
