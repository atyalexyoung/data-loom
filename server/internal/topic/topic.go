package topic

import (
	"fmt"
	"sync"

	"github.com/atyalexyoung/data-loom/server/internal/network"
)

type Topic struct {
	Name         string
	mu           sync.RWMutex
	Subscribers  map[*network.Client]bool
	Schemas      map[int]*TopicSchema
	LatestSchema int
}

func (t *Topic) Unsubscribe(client *network.Client) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.Subscribers[client]
	if !ok { // the client is not subscribed.
		return fmt.Errorf("cannot unsubscribe client from topic. client is not subscribed to topic. topic: %s, client: %s", t.Name, client.Id)
	}
	delete(t.Subscribers, client)
	return nil
}

func (t *Topic) IsClientSubscribed(client *network.Client) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.Subscribers[client]
	return ok
}

func (t *Topic) Subscribe(client *network.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Subscribers[client] = true
}

func (t *Topic) ListSubscribers() []*network.Client {
	t.mu.RLock()
	defer t.mu.RUnlock()

	clients := make([]*network.Client, 0, len(t.Subscribers))
	for c := range t.Subscribers {
		clients = append(clients, c)
	}
	return clients
}

// func (t *Topic) UpdateSchema(newSchema TopicSchema) int {
// 	t.mu.Lock()
// 	defer t.mu.Unlock()

// }

// GetLatestSchema will get the schema from the most recent version.
func (t *Topic) GetLatestSchema() (*TopicSchema, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	schema, ok := t.Schemas[t.LatestSchema]
	if !ok { // schema doesn't exist with latest schema number
		if len(t.Schemas) == 0 { // no schemas here, that's expected
			return nil, fmt.Errorf("no schemas found for topic. update schema.")
		} else { // schemas exist, but latest wasn't there, that's not good
			// find the real latest
			highest := 0
			for version, _ := range t.Schemas {
				if version > highest {
					version = highest
				}
			}

			if highest == 0 {
				t.LatestSchema = 0 // set version to zero
				return nil, fmt.Errorf("no schemas found. update schema")
			} else { // highest wasn't the real highest I guess. Log it for now

				// TODO: do something about this
				t.LatestSchema = highest
				schema, ok = t.Schemas[t.LatestSchema]
				if !ok {
					// TODO: clear out schema, something is very wrong.
					return nil, fmt.Errorf("no schemas found. update schema")
				}
				return schema, nil
			}
		}
	}
	return schema, nil
}

func (t *Topic) GetSchemaByVersion(versionNumber int) {
	t.mu.Lock()
	defer t.mu.Unlock()
}
