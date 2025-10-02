# Data Loom

Data Loom is a messaging system with persistence. 

The end-goal is to create a language agnostic pub/sub messaging system that has persistent storage and act as a repository to keep state for applications that communicate with each other.

> This project is for me to learn Go, some networking, SDK development etc. and NOT meant to be a stable product for use. While I am verifying some functionality with testing, I am not being as thorough with testing, logging, features nor documentation that would/should be required for a fully developed project for use in a production environment.

TLDR: This README provides a basic overview of the project. Other documentation for getting started, more details about various aspects of the project, and usage of the project under the /docs/ directory and includes:
- [Getting Started](/docs/getting-started.md)
- [API](/docs/api.md)
- [Server](/docs/server.md)
- [C-Sharp Client](/docs/csharp-client.md)

## Overview

> ⚠️ Note: This project is for learning purposes. It's not intended to be a stable production system. While some functionality is verified with testing, logging, and documentation are minimal, and the system may contain bugs.

### Topics
Similar to Kafka, topics act as the "topic of conversation" or "state variables" that clients can publish on or subscribe to.

Clients can subscribe to topics and receive messages in real time, with optional message persistence and retrieval (getting values without another client publishing to a topic).

#### Topic Schemas
Clients can register topics with a schema (a json structure) that acts as a type definition that all clients will need comply with in order to publish on a particular topic. The server will validate all messages being sent and ensure they comply with the latest topic schema. The goal for having schemas for topics is that in client side code you can have everything be typed and ensure the data you are passing and receiving conforms to a specific shape.

### Server
The server acts as a message broker, and also handles authorization, persistence, state and the schemas for individual "topics". 

As of now, it has some configuration and can use either Badger (embedded key-value in Go) or SQLite for persistence of state. It is planned to also have no persistence be an option, but for now the default in Badger.

### Clients and SDKs
Currently, the [C# SDK](/docs/csharp-client.md) is created but untested. There are stretch goals to create SDKs for Go and Typescript and have them be packages available to use that abstracts away all connection and messaging to the server.

Clients can connect to the server without needing the SDK and just connect via websockets and use the [API](/docs/api.md).

### Persistence
The server will also persist the latest state for all of the topics. This can help keep state if the server or any clients go down. There is a stretch goal to have storage be either the latest state or to keep a time-series based approach which would allow for "replaying" or analyzation of data over time.

Currently there is only [Badger](https://github.com/hypermodeinc/badger?tab=readme-ov-file#badgerdb) implementation for persistent storage with SQLite being implelemented but untested yet.

## Current State
- Core functionality implemented: topic creation, subscription, publishing, persistence.
- Some simple tests in place for message sending and receiving, along with simple stress tests, but not very robust yet.
- Not yet fully hardened for production use.
- Super basic auth with api key that is passed with inital Authorization header.

## Upcoming
- C# SDK is created with some testing + stress testing
- SQLite persistence is created but untested.
- Server pre-built binary for use without compilation and maybe a docker image.
- Better and more robust testing.
- Stretch goal of a Typescript SDK
- Configurable IP Address Binding
- Fixing persistence logging and ACKs when "no-persistence" is configured

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

This project uses [Badger](https://github.com/hypermodeinc/badger?tab=readme-ov-file#badgerdb), licensed under the Apache License 2.0.
