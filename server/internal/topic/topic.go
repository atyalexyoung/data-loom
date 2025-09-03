package topic

import (
	"fmt"
	"sync"

	"github.com/atyalexyoung/data-loom/server/internal/network"
)

type Topic struct {
	name         string
	mu           sync.RWMutex
	subscribers  map[*network.Client]bool
	schemas      map[int]*TopicSchema
	latestSchema int
}

func NewTopic(name string, schema map[string]any) *Topic {
	topic := &Topic{
		name:        name,
		schemas:     make(map[int]*TopicSchema),
		subscribers: make(map[*network.Client]bool),
		// LatestSchema default to 0
	}

	topic.schemas[0] = &TopicSchema{ // create new schema and add it to map
		Version: 0,
		Schema:  schema,
	}

	return topic
}

func (t *Topic) LatestSchemaVersion() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.latestSchema
}

func (t *Topic) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

func (t *Topic) Unsubscribe(client *network.Client) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.subscribers[client]
	if !ok { // the client is not subscribed.
		return fmt.Errorf("cannot unsubscribe client from topic. client is not subscribed to topic. topic: %s, client: %s", t.Name(), client.Id)
	}
	delete(t.subscribers, client)
	return nil
}

func (t *Topic) IsClientSubscribed(client *network.Client) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.subscribers[client]
	return ok
}

func (t *Topic) Subscribe(client *network.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subscribers[client] = true
}

func (t *Topic) ListSubscribers() []*network.Client {
	t.mu.RLock()
	defer t.mu.RUnlock()

	clients := make([]*network.Client, 0, len(t.subscribers))
	for c := range t.subscribers {
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
	schema, ok := t.schemas[t.latestSchema]
	if !ok { // schema doesn't exist with latest schema number
		if len(t.schemas) == 0 { // no schemas here, that's expected
			return nil, fmt.Errorf("no schemas found for topic. update schema.")
		} else { // schemas exist, but latest wasn't there, that's not good
			// find the real latest
			highest := 0
			for version, _ := range t.schemas {
				if version > highest {
					version = highest
				}
			}

			if highest == 0 {
				t.latestSchema = 0 // set version to zero
				return nil, fmt.Errorf("no schemas found. update schema")
			} else { // highest wasn't the real highest I guess. Log it for now

				// TODO: do something about this
				t.latestSchema = highest
				schema, ok = t.schemas[t.latestSchema]
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
