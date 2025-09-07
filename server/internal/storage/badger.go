package storage

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"sync"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStorage struct {
	database   *badger.DB
	writeQueue chan dbWriter
	mu         sync.Mutex
	closed     bool
}

type dbWriter struct {
	key   string
	value map[string]any
	errCh chan error
}

func NewBadgerStorage() *BadgerStorage {
	return &BadgerStorage{
		writeQueue: make(chan dbWriter, 5000),
	}
}

// OpenDatabase will handle logic for opening and setting up databse
func (s *BadgerStorage) Open(path string, ctx context.Context) error {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return err
	}
	s.database = db
	s.startWriter(ctx) // now we open, start.
	return nil
}

func (s *BadgerStorage) startWriter(ctx context.Context) {
	go func() {
		for {
			select {
			case write, ok := <-s.writeQueue:
				if !ok {
					return // queue closed
				}
				err := s.Put(write.key, write.value)
				if write.errCh != nil {
					write.errCh <- err
					log.Info("closing the write errCh")
					close(write.errCh)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Close will handle closing and cleaning up database instance
func (s *BadgerStorage) Close() error {
	s.mu.Lock()
	if !s.closed {
		close(s.writeQueue)
		s.closed = true
	}
	s.mu.Unlock()

	if s.database != nil {
		return s.database.Close()
	}
	return nil
}

// Put will set a key to a value that is passed in.
func (s *BadgerStorage) Put(key string, value map[string]any) error {

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = s.database.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *BadgerStorage) AsyncPut(key string, value map[string]any) chan error {
	ch := make(chan error, 1)

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		ch <- fmt.Errorf("storage is closed")
		close(ch)
		return ch
	}
	s.mu.Unlock()

	select {
	case s.writeQueue <- dbWriter{key: key, value: value, errCh: ch}:
		// queued successfully
	default:
		ch <- fmt.Errorf("write queue is full")
		close(ch)
	}
	return ch
}

// Get will retrieve the value of the supplied key
func (s *BadgerStorage) Get(key string) (map[string]any, error) {
	var result map[string]any

	err := s.database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return json.Unmarshal(val, &result)
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return result, nil
}

// Delete will delete a key, value pair from the database.
func (s *BadgerStorage) Delete(key string) error {
	err := s.database.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	if err != nil {
		return err
	}
	return nil
}
