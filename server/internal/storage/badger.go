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
	writeQueue chan dbWriteRequest
	mu         sync.Mutex
	closed     bool
}

type dbWriteRequest struct {
	key      string
	value    map[string]any
	errCh    chan error
	writeCtx context.Context
}

func NewBadgerStorage() *BadgerStorage {
	return &BadgerStorage{
		writeQueue: make(chan dbWriteRequest, 5000),
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

func (store *BadgerStorage) startWriter(ctx context.Context) {
	go func() {
		for {
			select {
			case writeReq, ok := <-store.writeQueue:
				if !ok {
					return // queue closed
				}

				// doing this
				select { // select context closed or proceed with write
				// if context closed, write error and continue with loop
				case <-writeReq.writeCtx.Done():
					if writeReq.errCh != nil {
						writeReq.errCh <- writeReq.writeCtx.Err()
					}
					continue
				default: // no cancellation, continue with operation
				}

				err := store.put(writeReq.key, writeReq.value)
				if writeReq.errCh != nil { // does this chan exist?
					writeReq.errCh <- err // give err to whoever sent this
					log.Info("closing the write errCh")
					close(writeReq.errCh)
				}

			case <-ctx.Done(): // if we get cancelled, stop the worker.
				return
			}
		}
	}()
}

// Close will handle closing and cleaning up database instance
func (store *BadgerStorage) Close() error {
	store.mu.Lock()
	if !store.closed {
		close(store.writeQueue)
		store.closed = true
	}
	store.mu.Unlock()

	if store.database != nil {
		return store.database.Close()
	}
	return nil
}

// Put will set a key to a value that is passed in.
func (store *BadgerStorage) put(key string, value map[string]any) error {

	byteData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	err = store.database.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), byteData)
	})
	if err != nil {
		return err
	}

	return nil
}

func (store *BadgerStorage) AsyncPut(ctx context.Context, key string, value map[string]any) chan error {
	returnChannel := make(chan error, 1)

	store.mu.Lock()
	if store.closed {
		store.mu.Unlock()
		returnChannel <- fmt.Errorf("storage is closed")
		close(returnChannel)
		return returnChannel
	}
	store.mu.Unlock()

	select {
	case store.writeQueue <- dbWriteRequest{key: key, value: value, errCh: returnChannel, writeCtx: ctx}:
		// queued successfully
	case <-ctx.Done():
		returnChannel <- ctx.Err()
		close(returnChannel)
	default:
		returnChannel <- fmt.Errorf("write queue is full")
		close(returnChannel)
	}
	return returnChannel
}

// Get will retrieve the value of the supplied key
func (store *BadgerStorage) Get(ctx context.Context, key string) (map[string]any, error) {
	var result map[string]any

	err := store.database.View(func(txn *badger.Txn) error {
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
func (store *BadgerStorage) Delete(ctx context.Context, key string) error {
	err := store.database.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	if err != nil {
		return err
	}
	return nil
}
