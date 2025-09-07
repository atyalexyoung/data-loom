# data-loom

This project is a messaging system with persistence. The end-goal is to create a language agnostic pub/sub messaging system that has persistent storage and act as a big state repository for applications that communicate with each other.

## Overview
This project is a WebSocket-based publish/subscribe system in Go. It allows clients to subscribe to topics and receive messages in real time, with optional message persistence and retrieval.
While there is only currently Badger implementation for persistent storage, SQLite will be added soon.
No SDKs are created yet, but it still could be used via raw websocket connection and sending the proper json messages to the server.

## Current State
- Core functionality implemented: topic creation, subscription, publishing, persistence (Badger only right now).
- Some simple tests in place for message sending and receiving, but not very robust yet.
- Not yet fully hardened for production use.
- Super basic auth with api key that is passed with inital Authorization header.

## Upcoming
- SDK creation for C#, Typescript and Go for easy use.
- Better documentation for usage of service.
- Better and more robust testing.
- More data persistance options (SQLite first, maybe more later if the need arises)

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
