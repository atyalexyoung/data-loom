package topic

import (
	"fmt"
	"sync"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Topic struct contains information about a topic.
// A topic represents a specific "topic of discussion" and a singular
// item that is to be published to, subscribed to, or have data pulled from.
// It is the metadata and definition around the data, and the schema will define
// what the actual data looks like. The name is how the topic is referenced, and
// the subscribers are the ones that care about this topic.
type Topic struct {
	name         string
	mu           sync.RWMutex
	subscribers  map[*network.Client]bool
	schemas      map[int]*TopicSchema
	latestSchema int
}

// NewTopic will intialize and return a ready to use Topic struct.
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

// LatestSchemaVersion will return the integer of the latest topic version.
func (t *Topic) LatestSchemaVersion() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.latestSchema
}

// Name will return the name of the topic.
func (t *Topic) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

// Unsubscribe will remove the client to the map of subscribers
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

// Subscribe will add the client to the map of subscribers
func (t *Topic) Subscribe(client *network.Client) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subscribers[client] = true
}

// IsClientSubscribed returns a bool if the client is in the map of subscribers.
func (t *Topic) IsClientSubscribed(client *network.Client) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	_, ok := t.subscribers[client]
	return ok
}

// ListSubscribers will get a list of network.Client type of all subscribers for the given topic
func (t *Topic) ListSubscribers() []*network.Client {
	t.mu.RLock()
	defer t.mu.RUnlock()

	clients := make([]*network.Client, 0, len(t.subscribers))
	for c := range t.subscribers {
		clients = append(clients, c)
	}
	return clients
}

// Update schema will update the schema of a topic, and return the new latest schema number.
func (t *Topic) UpdateSchema(schema map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.latestSchema++
	t.schemas[t.latestSchema] = &TopicSchema{
		Version: t.latestSchema,
		Schema:  schema,
	}
}

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

// GetSchemaByVersion will get the schema for the topic of the given version interger.
func (t *Topic) GetSchemaByVersion(versionNumber int) (*TopicSchema, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	schema, ok := t.schemas[t.latestSchema]
	if !ok {
		return nil, fmt.Errorf("cannot get schema version %d. Version doesn't exist", versionNumber)
	}

	return schema, nil
}

func (t *Topic) Publish(sender *network.Client, value []byte) []*network.Client {
	t.mu.Lock()
	defer t.mu.Unlock()

	failedClients := make([]*network.Client, 0)

	// publish to all subscribers
	for client := range t.subscribers {
		if err := client.SendJSON(value); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				failedClients = append(failedClients, client)
			}
			// TODO: add failure count to client failure
			log.Println("Error when writing json to client: ", client.Id)
		}
	}
	return failedClients
}
