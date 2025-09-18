package storage

import (
	"context"
	"time"

	"github.com/atyalexyoung/data-loom/server/internal/config"
)

type dbWriteRequest struct {
	key       string
	value     map[string]any
	errCh     chan error
	writeCtx  context.Context
	timestamp time.Time
}

// Storage is an interface for any storage that will be used.
type Storage interface {

	// OpenDatabase will handle logic for opening and setting up databse
	Open(path string, ctx context.Context) error

	// Close will handle closing and cleaning up database instance
	Close() error

	// AsyncPut will set a key to a value that is passed in.
	AsyncPut(ctx context.Context, key string, value map[string]any, timestamp time.Time) chan error

	// Get will retrieve the value of the supplied key
	Get(ctx context.Context, key string) (map[string]any, error)

	// Delete will delete a key, value pair from the database.
	Delete(ctx context.Context, key string) error
}

// NewStorage takes the configuration and returns the storage type that is specified.
func NewStorage(cfg *config.Config, ctx context.Context) (Storage, error) {
	switch cfg.StorageType {
	case "badger":
		s := NewBadgerStorage()
		if err := s.Open(cfg.StoragePath, ctx); err != nil {
			return nil, err
		}
		return s, nil
	case "sqlite":
		s := NewSqliteStorage()
		if err := s.Open(cfg.StoragePath, ctx); err != nil {
			return nil, err
		}
		return s, nil
	default: // for now during dev just use badger so I don't have to set up the actual stuff
		s := NewBadgerStorage()
		if err := s.Open(cfg.StoragePath, ctx); err != nil {
			return nil, err
		}
		return s, nil
		//return nil, fmt.Errorf("unknown storage type: %s", cfg.StorageType)
	}
}
