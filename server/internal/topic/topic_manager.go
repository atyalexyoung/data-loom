package topic

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/storage"
)

type TopicManager interface {
	Subscribe(topicName string, client *network.Client) error
	Unsubscribe(topicName string, client *network.Client) error
	ListSubscribersForTopic(topicName string) ([]*network.Client, error)
	UnsubscribeAll(client *network.Client)
	Publish(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, errChan chan error) error
	SendWithoutSave(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, errChan chan error) error
	Get(ctx context.Context, topicName string) (map[string]any, error)
	RegisterTopic(topicName string, schema map[string]any) (*Topic, error)
	UnregisterTopic(ctx context.Context, topicName string) error
	ListTopics() ([]*Topic, error)
	UpdateSchema(topicName string, schema map[string]any) error
	NextFailedClient() (*network.Client, bool)
	IsSchemaMatch(topicName string, schema map[string]any) (bool, error)
}

// topicManager holds a map of the key for a key-value pair and the client that is subscribed to that key.
type topicManager struct {
	mu            sync.RWMutex
	topics        map[string]*Topic
	db            storage.Storage
	failedClients chan *network.Client
}

func NewTopicManager(storage storage.Storage) TopicManager {
	return &topicManager{
		topics:        make(map[string]*Topic),
		db:            storage,
		failedClients: make(chan *network.Client, 100),
	}
}

func (tm *topicManager) NextFailedClient() (*network.Client, bool) {
	client, ok := <-tm.failedClients
	return client, ok
}

// will increment the amount of failures for a client in the
func (tm *topicManager) markClientFailed(c *network.Client) {
	tm.failedClients <- c
}

// Subscribe checks if the topic
func (tm *topicManager) Subscribe(topicName string, client *network.Client) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topic, exists := tm.topics[topicName]
	if !exists { // if topic doesn't exist, just let the user know
		return fmt.Errorf("topic doesn't exist for %s", topicName)
	}

	topic.Subscribe(client)
	return nil
}

// Unsubscribe removes a client from the subscription list for a given topic name.
func (tm *topicManager) Unsubscribe(topicName string, client *network.Client) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topic, ok := tm.topics[topicName]
	if !ok { // the topic doesn't exist to unsubscribe from, let user know
		return fmt.Errorf("cannot unsubscribe client from topic. topic doesn't exits. topic: %s, client: %s", topicName, client.Id)
	}
	return topic.Unsubscribe(client)
}

// ListSubscribersForTopic returns a copy of the list of all clients that are subscribed to a given topic name.
func (tm *topicManager) ListSubscribersForTopic(topicName string) ([]*network.Client, error) {
	topic, ok := tm.topics[topicName]
	if !ok {
		return nil, fmt.Errorf("cannot get subscribers for topic. topic doesn't exist. Topic: %s", topicName)
	}
	return topic.ListSubscribers(), nil
}

// UnsubscribeAll removes a client from all topics.
func (tm *topicManager) UnsubscribeAll(client *network.Client) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, topic := range tm.topics {
		err := topic.Unsubscribe(client)
		if err == nil { // if we get error, the client wasn't subscribed to topic
			log.Printf("Unsubscribed client: %s from topic: %s", client.Id, topic.Name())
		}
	}
}

// sendTopic will send the value passed in for a given topic to all the subscribers of that topic.
func (tm *topicManager) sendTopic(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, persist bool, errCh chan error) error {
	// get topic from tm and unlock
	tm.mu.RLock()
	topic, ok := tm.topics[msg.Topic]
	tm.mu.RUnlock()

	if !ok { // couldn't get topic, I guess it doesn't exist
		return fmt.Errorf("publish failed. Topic doesn't exist. Topic: %s", msg.Topic)
	}

	var dbErrChan chan error
	if persist { // if it's supposed to be persisted, then persist
		time := time.Now().UTC()

		var valueString string

		if raw, err := json.Marshal(value); err != nil {
			valueString = fmt.Sprintf("marshal_error: %v, fallback=%#v", err, value)
		} else {
			valueString = string(raw)
		}
		log.WithFields(log.Fields{
			"sender_id":  sender.Id,
			"value":      valueString,
			"action":     msg.Action,
			"message_id": msg.Id,
			"topic":      msg.Topic,
			"time":       time,
		}).Info("persisting message")

		dbErrChan = tm.db.AsyncPut(ctx, msg.Topic, value, time)
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("Could not marshal json data.")
	}
	outboundMessage := &network.WebSocketMessage{
		Id:     msg.Id,
		Action: msg.Action,
		Topic:  msg.Topic,
		Data:   raw,
	}
	failedClients := topic.Publish(sender, outboundMessage)

	for _, client := range failedClients {
		tm.markClientFailed(client)
	}

	// respond to client with errors if needed
	if dbErrChan != nil && errCh != nil {
		go func() {
			defer close(errCh)

			select {
			case err := <-dbErrChan:
				if err != nil {
					errCh <- fmt.Errorf("database error: %w", err)
				}
			case <-time.After(2 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for database ack")
			}
		}()
	} else if errCh != nil {
		close(errCh) // if no persistence, just close
	}

	return nil
}

// Publish will send the JSON of the message to all clients subscribed to the topic
func (tm *topicManager) Publish(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, errChan chan error) error {
	return tm.sendTopic(ctx, msg, sender, value, true, errChan)
}

// SendWithoutSave will publish a value to a topic, but not persist that data to storage.
func (tm *topicManager) SendWithoutSave(ctx context.Context, msg network.WebSocketMessage, sender *network.Client, value map[string]any, errChan chan error) error {
	return tm.sendTopic(ctx, msg, sender, value, false, errChan)
}

// Get will retrieve the current value for a given topic
func (tm *topicManager) Get(ctx context.Context, topicName string) (map[string]any, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topic, ok := tm.topics[topicName]
	if !ok {
		return nil, fmt.Errorf("couldn't get value for topic. topic doesn't exist. topic: %s", topicName)
	}

	value, err := tm.db.Get(ctx, topic.Name())
	if err != nil {
		return nil, fmt.Errorf("couldn't get value for topic with error: %v", err)
	}
	return value, nil
}

// RegisterTopic takes a topic name and schema for the topic and will add it to list of topics.
// This will create a schema of version 0 for the topic. Returns error if the topic already exists
func (tm *topicManager) RegisterTopic(topicName string, schema map[string]any) (*Topic, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	currentTopic, ok := tm.topics[topicName]
	if ok { // if we get a topic, it already exists
		curretSchema, err := currentTopic.GetLatestSchema()
		if err == nil { // WE DID GET THE LATEST SCHEMA
			if schemasMatch(curretSchema.Schema, schema) {
				return currentTopic, nil
			} else { // schemas don't match, return error
				return nil, fmt.Errorf("cannot register topic, topic already exists with different schema. Try updating schema")
			}
		} // else we couldn't get the latest schema, update the current topics schema.
		currentTopic.UpdateSchema(schema)
		return currentTopic, nil

	} // else we didn't get a topic so create new one.
	topic := NewTopic(topicName, schema)
	tm.topics[topic.Name()] = topic // add new topic to topic manager

	return topic, nil
}

// schemasMatch will convert two map[string]any tol json and compare them to see if they are the same.
func schemasMatch(a, b map[string]any) bool {
	aBytes, err1 := json.Marshal(a)
	bBytes, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(aBytes) == string(bBytes)
}

// UnregisterTopic takes name of topic to unregister and removes it from the topics.
// returns error if topic doesn't exist.
func (tm *topicManager) UnregisterTopic(ctx context.Context, topicName string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, ok := tm.topics[topicName]
	if !ok {
		return fmt.Errorf("cannot unregister topic. topic doesn't exist with name: %s", topicName)
	}

	delete(tm.topics, topicName) // delete the key-value in the map

	if err := tm.db.Delete(ctx, topicName); err != nil {
		return fmt.Errorf("Topic deleted but unable to delete from persistent storage with err: %v", err)
	}

	return nil
}

// ListTopics will retreive all topics that are currently being used.
func (tm *topicManager) ListTopics() ([]*Topic, error) {

	// get topics copy and unlock manager
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topicsCopy := make([]*Topic, 0, len(tm.topics))
	for _, t := range tm.topics {
		topicsCopy = append(topicsCopy, t)
	}

	return topicsCopy, nil
}

func (tm *topicManager) UpdateSchema(topicName string, schema map[string]any) error {
	tm.mu.Lock()
	topic, ok := tm.topics[topicName]
	tm.mu.Unlock()

	if !ok {
		return fmt.Errorf("cannot update schema for topic %s. Topic doesn't exist", topicName)
	}
	topic.UpdateSchema(schema)
	return nil
}

// getLatestSchemaForTopic does what it says it will do. Gets the latest schema for a given topic.
func (tm *topicManager) getLatestSchemaForTopic(topicName string) (*TopicSchema, error) {
	tm.mu.Lock()
	topic, ok := tm.topics[topicName]
	tm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("could not get topic by name: %s", topicName)
	}

	schema, err := topic.GetLatestSchema()
	if err != nil {
		return nil, fmt.Errorf("could not get schema for topic with name: %s", topicName)
	}

	return schema, nil
}

// IsSchemaMatch will compare the current schema for a topic and the schema passed in to check
// if the schema matches the current schema
func (tm *topicManager) IsSchemaMatch(topicName string, schema map[string]any) (bool, error) {

	currentSchema, err := tm.getLatestSchemaForTopic(topicName)
	if err != nil { // can't get this topic's schema, that's no good.
		return false, err
	}

	if !schemasMatch(schema, currentSchema.Schema) { // schemas don't match, get with it yo
		return false, fmt.Errorf("schema doesn't match topics current schema")
	}

	return true, nil
}
