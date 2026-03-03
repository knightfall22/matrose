package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
	"github.com/knightfall22/matrose/task"
)

type TaskStore struct {
	Db       *bolt.DB
	DbFile   string
	FileMode os.FileMode
	Bucket   string
}

func NewTaskStore(file string, mode os.FileMode, bucket string) (*TaskStore, error) {
	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, err
	}

	t := TaskStore{
		Db:       db,
		DbFile:   file,
		FileMode: mode,
		Bucket:   bucket,
	}

	err = t.CreateBucket()
	if err != nil {
		log.Printf("bucket already exists, will use it instead of creating new one")
	}

	return &t, nil
}

func (ts *TaskStore) CreateBucket() error {
	return ts.Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(ts.Bucket))
		if err != nil {
			return fmt.Errorf("create bucket %s: %s", ts.Bucket, err)
		}
		return nil
	})
}

func (ts *TaskStore) Count() (int, error) {
	taskCount := 0
	err := ts.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ts.Bucket))

		b.ForEach(func(k, v []byte) error {
			taskCount++
			return nil
		})

		return nil
	})
	if err != nil {
		return -1, err
	}

	return taskCount, nil
}

func (ts *TaskStore) Close() {
	ts.Db.Close()
}

func (ts *TaskStore) Put(key string, value interface{}) error {
	return ts.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ts.Bucket))

		buf, err := json.Marshal(value.(*task.Task))
		if err != nil {
			return err
		}

		err = b.Put([]byte(key), buf)
		if err != nil {
			return err
		}

		return nil
	})
}

func (ts *TaskStore) Get(key string) (interface{}, error) {
	var task task.Task
	err := ts.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ts.Bucket))

		t := b.Get([]byte(key))
		if t == nil {
			return fmt.Errorf("task %v not found", key)
		}

		err := json.Unmarshal(t, &task)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (ts *TaskStore) List() (interface{}, error) {
	var tasks []*task.Task
	err := ts.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(ts.Bucket))
		b.ForEach(func(k, v []byte) error {
			var task task.Task
			err := json.Unmarshal(v, &task)
			if err != nil {
				return err
			}
			tasks = append(tasks, &task)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

type EventStore struct {
	Db       *bolt.DB
	DbFile   string
	FileMode os.FileMode
	Bucket   string
}

func NewEventStore(file string, mode os.FileMode, bucket string) (*EventStore, error) {
	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, err
	}

	t := EventStore{
		Db:       db,
		DbFile:   file,
		FileMode: mode,
		Bucket:   bucket,
	}

	err = t.CreateBucket()
	if err != nil {
		log.Printf("bucket already exists, will use it instead of creating new one")
	}

	return &t, nil
}

func (e *EventStore) CreateBucket() error {
	return e.Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(e.Bucket))
		if err != nil {
			return fmt.Errorf("create bucket %s: %s", e.Bucket, err)
		}
		return nil
	})
}

func (e *EventStore) Count() (int, error) {
	eventCount := 0
	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))

		b.ForEach(func(k, v []byte) error {
			eventCount++
			return nil
		})

		return nil
	})
	if err != nil {
		return -1, err
	}

	return eventCount, nil
}

func (e *EventStore) Close() {
	e.Db.Close()
}

func (e *EventStore) Put(key string, value interface{}) error {
	return e.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))

		buf, err := json.Marshal(value.(*task.TaskEvent))
		if err != nil {
			return err
		}

		err = b.Put([]byte(key), buf)
		if err != nil {
			return err
		}

		return nil
	})
}

func (e *EventStore) Get(key string) (interface{}, error) {
	var task task.TaskEvent
	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))

		t := b.Get([]byte(key))
		if t == nil {
			return fmt.Errorf("task %v not found", key)
		}

		err := json.Unmarshal(t, &task)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (e *EventStore) List() (interface{}, error) {
	var tasks []*task.TaskEvent
	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		b.ForEach(func(k, v []byte) error {
			var task task.TaskEvent
			err := json.Unmarshal(v, &task)
			if err != nil {
				return err
			}
			tasks = append(tasks, &task)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}
