package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

type SqliteStorage struct {
	db         *sql.DB
	writeQueue chan dbWriteRequest
	mu         sync.Mutex
	closed     bool
}

func (s *SqliteStorage) Open(path string, ctx context.Context) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	sqlStmt := `
		CREATE TABLE IF NOT EXISTS messages (
			topicName TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			data BLOB NOT NULL,
			PRIMARY KEY (topicName, timestamp)
		);

		CREATE INDEX IF NOT EXISTS idx_topicName ON messages(topicName);
		CREATE INDEX IF NOT EXISTS idx_timestamp ON messages(timestamp);
	`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		db.Close()
		return err
	}

	s.startWriter(ctx) // now we open, start.

	return nil
}

func (store *SqliteStorage) startWriter(ctx context.Context) {
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

				err := store.put(writeReq.writeCtx, writeReq.key, writeReq.value, writeReq.timestamp)
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
func (s *SqliteStorage) Close() error {

	s.mu.Lock()
	if !s.closed {
		close(s.writeQueue)
		s.closed = true
	}
	s.mu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Put will set a key to a value that is passed in.
func (s *SqliteStorage) put(ctx context.Context, key string, value map[string]any, timestamp time.Time) error {

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// TODO: maybe handle error from SQL on collision instead of direct replace.
	const insertStatement = `
		INSERT OR REPLACE INTO messages (topicName, timestamp, data)
		VALUES (?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, insertStatement, key, timestamp, data)
	return err
}

func (s *SqliteStorage) AsyncPut(ctx context.Context, key string, value map[string]any, timestamp time.Time) chan error {
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
	case s.writeQueue <- dbWriteRequest{key: key, value: value, errCh: ch}:
		// queued successfully
	default:
		ch <- fmt.Errorf("write queue is full")
		close(ch)
	}
	return ch
}

// Get will retrieve the value of the supplied key
func (store *SqliteStorage) Get(ctx context.Context, key string) (map[string]any, error) {

	const query = `
	SELECT data FROM messages
		WHERE topicName = ? AND timestamp = ?
	`

	var rawData []byte
	err := store.db.QueryRowContext(ctx, query, key).Scan(rawData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(rawData, &result); err != nil {

	}

	return result, nil
}

// Delete will delete a key, value pair from the database.
func Delete(ctx context.Context, key string) error {
	return nil
}
