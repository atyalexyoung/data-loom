package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	_ "modernc.org/sqlite"
)

type SqliteStorage struct {
	db         *sql.DB
	writeQueue chan dbWriter
	mu         sync.Mutex
	closed     bool
}

func Open(path string, ctx context.Context) error {
	db, err := sql.Open("sqlite", "./test.db")
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        name TEXT
    );`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}

	return nil
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
func Put(key string, value map[string]any) error {

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fmt.Println(data)

	return nil
}

func (s *SqliteStorage) AsyncPut(key string, value map[string]any) chan error {
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
func Get(key string) (map[string]any, error) {
	var result map[string]any

	return result, nil
}

// Delete will delete a key, value pair from the database.
func Delete(key string) error {
	return nil
}
