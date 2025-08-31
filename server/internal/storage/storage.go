package storage

import (
	"github.com/atyalexyoung/data-loom/server/internal/config"
)

// Storage is an interface for any storage that will be used.
type Storage interface {

	// OpenDatabase will handle logic for opening and setting up databse
	Open(path string) error

	// Close will handle closing and cleaning up database instance
	Close() error

	// Put will set a key to a value that is passed in.
	Put(key string, value []byte) error

	// Get will retrieve the value of the supplied key
	Get(key string) ([]byte, error)

	// Delete will delete a key, value pair from the database.
	Delete(key string) error
}

// NewStorage takes the configuration and returns the storage type that is specified.
func NewStorage(cfg *config.Config) (Storage, error) {
	switch cfg.StorageType {
	case "badger":
		s := &BadgerStorage{}
		if err := s.Open(cfg.StoragePath); err != nil {
			return nil, err
		}
		return s, nil
	// case "sqlite":
	// 	s := &storage.SQLiteStorage{}
	// 	if err := s.Open(cfg.StoragePath); err != nil {
	// 		return nil, err
	// 	}
	// 	return s, nil
	default: // for now during dev just use badger so I don't have to set up the actual stuff
		s := &BadgerStorage{}
		if err := s.Open(cfg.StoragePath); err != nil {
			return nil, err
		}
		return s, nil
		//return nil, fmt.Errorf("unknown storage type: %s", cfg.StorageType)
	}
}
