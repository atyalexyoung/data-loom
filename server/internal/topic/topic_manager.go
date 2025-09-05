package topic

import (
	"encoding/json"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/storage"
)

type TopicManager interface {
	Subscribe(topicName string, client *network.Client) error
	Unsubscribe(topicName string, client *network.Client) error
	ListSubscribersForTopic(topicName string) ([]*network.Client, error)
	UnsubscribeAll(client *network.Client)
	Publish(topicName string, sender *network.Client, value map[string]any) error
	SendWithoutSave(topicName string, sender *network.Client, value map[string]any) error
	Get(topicName string) ([]byte, error)
	RegisterTopic(topicName string, schema map[string]any) (*Topic, error)
	UnregisterTopic(topicName string) error
	ListTopics() ([]*Topic, error)
	UpdateSchema(topicName string, schema map[string]any) error
	NextFailedClient() (*network.Client, bool)
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
func (tm *topicManager) sendTopic(topicName string, sender *network.Client, value map[string]any, isOnlySend bool) error {
	// get topic from tm and unlock
	tm.mu.RLock()
	topic, ok := tm.topics[topicName]
	tm.mu.RUnlock()

	if !ok { // couldn't get topic, I guess it doesn't exist
		return fmt.Errorf("publish failed. Topic doesn't exist. Topic: %s", topicName)
	}

	if !isOnlySend { // if it's supposed to be persisted, then persist
		tm.db.Put(topicName, value)
	}

	failedClients := topic.Publish(sender, value)

	for _, client := range failedClients {
		tm.markClientFailed(client)
	}
	return nil
}

// Publish will send the JSON of the message to all clients subscribed to the topic
func (tm *topicManager) Publish(topicName string, sender *network.Client, value map[string]any) error {
	return tm.sendTopic(topicName, sender, value, false)
}

// SendWithoutSave will publish a value to a topic, but not persist that data to storage.
func (tm *topicManager) SendWithoutSave(topicName string, sender *network.Client, value map[string]any) error {
	return tm.sendTopic(topicName, sender, value, true)
}

// Get will retrieve the current value for a given topic
func (tm *topicManager) Get(topicName string) ([]byte, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topic, ok := tm.topics[topicName]
	if !ok {
		return nil, fmt.Errorf("couldn't get value for topic. topic doesn't exist. topic: %s", topicName)
	}

	value, err := tm.db.Get(topic.Name())
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
				return nil, fmt.Errorf("cannot register topic, topic already exists with different schema. Try updating schema.")
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
func (tm *topicManager) UnregisterTopic(topicName string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, ok := tm.topics[topicName]
	if !ok {
		return fmt.Errorf("cannot unregister topic. topic doesn't exist with name: %s", topicName)
	}

	delete(tm.topics, topicName) // delete the key-value in the map

	if err := tm.db.Delete(topicName); err != nil {
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
		return fmt.Errorf("Cannot update schema for topic %s. Topic doesn't exist", topicName)
	}
	topic.UpdateSchema(schema)
	return nil
}
