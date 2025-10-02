package topic

import (
	"fmt"

	"github.com/atyalexyoung/data-loom/server/internal/logging"
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
	mu           logging.DebugRWMutex
	subscribers  map[*network.Client]bool
	schemas      map[int]*TopicSchema
	latestSchema int
}

// TopicSchema defines the data that is held to define a schema for a topic
// version number is the current version for the schema and the Schema is
// the json structure of the schema.
type TopicSchema struct {
	Version int
	Schema  map[string]any
}

// NewTopic will intialize and return a ready to use Topic struct.
func NewTopic(name string, schema map[string]any) *Topic {
	topic := &Topic{
		name:        name,
		schemas:     make(map[int]*TopicSchema),
		subscribers: make(map[*network.Client]bool),
		mu:          *logging.NewDebugRWMutex("Topic: " + name),
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
	t.mu.RLock("LatestSchemaVersion")
	defer t.mu.RUnlock("LatestSchemaVersion")
	return t.latestSchema
}

// NameWithLock will return the name of the topic.
func (t *Topic) NameWithLock() string {
	t.mu.RLock("Name")
	defer t.mu.RUnlock("Name")
	return t.name
}

// Unsubscribe will remove the client to the map of subscribers
func (t *Topic) Unsubscribe(client *network.Client) error {
	t.mu.Lock("Unsubscribe")
	defer t.mu.Unlock("Unsubscribe")
	_, ok := t.subscribers[client]
	if !ok { // the client is not subscribed.
		return fmt.Errorf("cannot unsubscribe client from topic. client is not subscribed to topic. topic: %s, client: %s", t.name, client.Id)
	}
	delete(t.subscribers, client)
	return nil
}

// Subscribe will add the client to the map of subscribers
func (t *Topic) Subscribe(client *network.Client) {
	t.mu.Lock("Subscribe")
	defer t.mu.Unlock("Subscribe")
	t.subscribers[client] = true
}

// IsClientSubscribed returns a bool if the client is in the map of subscribers.
func (t *Topic) IsClientSubscribed(client *network.Client) bool {
	t.mu.RLock("IsClientSubscribed")
	defer t.mu.RUnlock("IsClientSubscribed")
	_, ok := t.subscribers[client]
	return ok
}

// ListSubscribers will get a list of network.Client type of all subscribers for the given topic
func (t *Topic) ListSubscribers() []*network.Client {
	t.mu.RLock("ListSubscribers")
	defer t.mu.RUnlock("ListSubscribers")

	clients := make([]*network.Client, 0, len(t.subscribers))
	for c := range t.subscribers {
		clients = append(clients, c)
	}
	return clients
}

// Update schema will update the schema of a topic, and return the new latest schema number.
func (t *Topic) UpdateSchema(schema map[string]any) {
	t.mu.Lock("UpdateSchema")
	defer func() {
		log.WithFields(log.Fields{"method": "UpdateSchema", "topic": t.name}).Trace("updated topic schema")
		t.mu.Unlock("UpdateSchema")
	}()

	t.latestSchema++
	t.schemas[t.latestSchema] = &TopicSchema{
		Version: t.latestSchema,
		Schema:  schema,
	}
}

// GetLatestSchema will get the schema from the most recent version.
func (t *Topic) GetLatestSchema() (*TopicSchema, error) {
	t.mu.Lock("GetLatestSchema")
	defer t.mu.Unlock("GetLatestSchema")

	// If no schemas exist at all
	if len(t.schemas) == 0 {
		return nil, fmt.Errorf("no schemas found for topic, update schema")
	}

	// Try the cached latest
	if schema, ok := t.schemas[t.latestSchema]; ok {
		return schema, nil
	}

	// If cached latest is wrong, find the real latest
	highest := 0
	for v := range t.schemas {
		if v > highest {
			highest = v
		}
	}

	t.latestSchema = highest
	if schema, ok := t.schemas[highest]; ok {
		return schema, nil
	}

	// This should never happen if len > 0
	return nil, fmt.Errorf("schema map is corrupt, no latest schema found")
}

// GetSchemaByVersion will get the schema for the topic of the given version interger.
func (t *Topic) GetSchemaByVersion(versionNumber int) (*TopicSchema, error) {
	t.mu.Lock("GetSchemaByVersion")
	defer t.mu.Unlock("GetSchemaByVersion")

	schema, ok := t.schemas[t.latestSchema]
	if !ok {
		return nil, fmt.Errorf("cannot get schema version %d. Version doesn't exist", versionNumber)
	}

	return schema, nil
}

func (t *Topic) Publish(sender *network.Client, msg *network.WebSocketMessage) []*network.Client {
	t.mu.Lock("Publish")
	defer t.mu.Unlock("Publish")

	failedClients := make([]*network.Client, 0)

	// publish to all subscribers
	for client := range t.subscribers {
		if err := client.SendJSON(msg); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				failedClients = append(failedClients, client)
			}
			// TODO: add failure count to client failure
			log.Println("Error when writing json to client: ", client.Id)
		}
	}
	return failedClients
}
