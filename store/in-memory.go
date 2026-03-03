package store

import (
	"fmt"

	"github.com/knightfall22/matrose/task"
)

type InMemoryTaskStore struct {
	DB map[string]*task.Task
}

func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		DB: make(map[string]*task.Task),
	}
}

func (i *InMemoryTaskStore) Put(key string, value any) error {
	v, ok := value.(*task.Task)
	if !ok {
		return fmt.Errorf("value %v is not a task.Task type", value)
	}

	i.DB[key] = v
	return nil
}

func (i *InMemoryTaskStore) Get(key string) (any, error) {
	t, ok := i.DB[key]
	if !ok {
		return nil, fmt.Errorf("task with key %s does not exist", key)
	}

	return t, nil
}

func (i *InMemoryTaskStore) List() (any, error) {
	dbLen := len(i.DB)
	output := make([]*task.Task, dbLen)
	idx := 0

	for _, t := range i.DB {
		output[idx] = t
		idx++
	}

	return output, nil
}

func (i *InMemoryTaskStore) Count() (int, error) {
	dbLen := len(i.DB)

	return dbLen, nil
}

type InMemoryTaskEventStore struct {
	DB map[string]*task.TaskEvent
}

func NewInMemoryTaskEventStore() *InMemoryTaskEventStore {
	return &InMemoryTaskEventStore{
		DB: make(map[string]*task.TaskEvent),
	}
}

func (i *InMemoryTaskEventStore) Put(key string, value any) error {
	v, ok := value.(*task.TaskEvent)
	if !ok {
		return fmt.Errorf("value %v is not a task.Task type", value)
	}

	i.DB[key] = v
	return nil
}

func (i *InMemoryTaskEventStore) Get(key string) (any, error) {
	t, ok := i.DB[key]
	if !ok {
		return nil, fmt.Errorf("task with key %s does not exist", key)
	}

	return t, nil
}

func (i *InMemoryTaskEventStore) List() (any, error) {
	dbLen := len(i.DB)
	output := make([]*task.TaskEvent, dbLen)
	idx := 0

	for _, t := range i.DB {
		output[idx] = t
		idx++
	}

	return output, nil
}

func (i *InMemoryTaskEventStore) Count() (int, error) {
	dbLen := len(i.DB)

	return dbLen, nil
}
