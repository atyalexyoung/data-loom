package storage

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

type NullStorage struct{}

func NewNullStorage() *NullStorage {
	return &NullStorage{}
}

func (n *NullStorage) Open(path string, ctx context.Context) error {
	log.Debugf("[NullStorage] Open called with path: %s", path)
	return nil
}

func (n *NullStorage) Close() error {
	log.Debug("[NullStorage] Close called")
	return nil
}

func (n *NullStorage) AsyncPut(ctx context.Context, key string, value map[string]any, timestamp time.Time) chan error {
	log.WithFields(log.Fields{
		"key":       key,
		"value":     value,
		"timestamp": timestamp,
	}).Debug("[NullStorage] AsyncPut called")
	ch := make(chan error, 1)
	ch <- nil
	close(ch)
	return ch
}

func (n *NullStorage) Get(ctx context.Context, key string) (map[string]any, error) {
	log.Debugf("[NullStorage] Get called for key: %s", key)
	return nil, nil
}

func (n *NullStorage) Delete(ctx context.Context, key string) error {
	log.Debugf("[NullStorage] Delete called for key: %s", key)
	return nil
}
