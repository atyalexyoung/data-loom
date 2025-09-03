package topic

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/atyalexyoung/data-loom/server/internal/network"
	"github.com/atyalexyoung/data-loom/server/internal/storage"
	"github.com/gorilla/websocket"
)

// TopicManager holds a map of the key for a key-value pair and the client that is subscribed to that key.
type TopicManager struct {
	mu     sync.RWMutex
	topics map[string]*Topic
	db     storage.Storage
}

type TopicSchema struct {
	Version int
	Schema  map[string]any
}

func NewTopicManager(storage storage.Storage) *TopicManager {
	return &TopicManager{
		topics: make(map[string]*Topic),
		db:     storage,
	}
}

// Subscribe checks if the topic
func (tm *TopicManager) Subscribe(topicName string, client *network.Client) error {
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
func (tm *TopicManager) Unsubscribe(topicName string, client *network.Client) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topic, ok := tm.topics[topicName]
	if !ok { // the topic doesn't exist to unsubscribe from, let user know
		return fmt.Errorf("cannot unsubscribe client from topic. topic doesn't exits. topic: %s, client: %s", topicName, client.Id)
	}
	return topic.Unsubscribe(client)
}

// ListSubscribersForTopic returns a copy of the list of all clients that are subscribed to a given topic name.
func (tm *TopicManager) ListSubscribersForTopic(topicName string) ([]*network.Client, error) {
	topic, ok := tm.topics[topicName]
	if !ok {
		return nil, fmt.Errorf("cannot get subscribers for topic. topic doesn't exist. Topic: %s", topicName)
	}
	return topic.ListSubscribers(), nil
}

// UnsubscribeAll removes a client from all topics.
func (tm *TopicManager) UnsubscribeAll(client *network.Client) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, topic := range tm.topics {
		err := topic.Unsubscribe(client)
		if err == nil { // if we get error, the client wasn't subscribed to topic
			log.Printf("Unsubscribed client: %s from topic: %s", client.Id, topic.Name())
		}
	}
}

// Publish will send the JSON of the message to all clients subscribed to the topic
func (tm *TopicManager) Publish(topicName string, sender *network.Client, value []byte) error {

	// get topic from tm and unlock
	tm.mu.Lock()
	topic, ok := tm.topics[topicName]
	tm.mu.Unlock()

	if !ok { // couldn't get topic, I guess it doesn't exist
		return fmt.Errorf("publish failed. Topic doesn't exist. Topic: %s", topicName)
	}

	// get the subscribers to send to and unlock the topic
	topic.mu.Lock()
	subsribers := make([]*network.Client, 0, len(topic.subscribers))
	for client := range topic.subscribers {
		if client != sender {
			subsribers = append(subsribers, client)
		}
	}
	topic.mu.Unlock()

	tm.db.Put(topicName, value) // persist to db, let them worry about async or anything like that

	// publish to all subscribers
	for _, client := range subsribers {
		if err := client.SendJSON(value); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				go tm.UnsubscribeAll(client)
			}
			// TODO: add failure count to client failure
			log.Println("Error when writing json to client: ", client.Id)
		}
	}
	return nil
}

// Get will retrieve the current value for a given topic
func (tm *TopicManager) Get(topicName string) ([]byte, error) {
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
func (tm *TopicManager) RegisterTopic(topicName string, schema map[string]any) (*Topic, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, ok := tm.topics[topicName]
	if ok { // if we get a topic, it already exists
		return nil, fmt.Errorf("cannot register topic. topic already exists. consider updating topic")
	}
	topic := NewTopic(topicName, schema)

	// add new topic to topic manager
	tm.topics[topic.Name()] = topic

	return topic, nil
}

// UnregisterTopic takes name of topic to unregister and removes it from the topics.
// returns error if topic doesn't exist.
func (tm *TopicManager) UnregisterTopic(topicName string) error {
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
func (tm *TopicManager) ListTopics() ([]*Topic, error) {

	// get topics copy and unlock manager
	tm.mu.Lock()
	defer tm.mu.Unlock()

	topicsCopy := make([]*Topic, 0, len(tm.topics))
	for _, t := range tm.topics {
		topicsCopy = append(topicsCopy, t)
	}

	return topicsCopy, nil
}
