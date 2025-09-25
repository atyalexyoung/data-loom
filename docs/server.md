# Server

This documentation is planned to be expanded upon.

## Features
- Message brokering via WebSockets
- Persistence using Badger or SQLite
- Topic schemas with validation
- Basic API key authentication

## Configuration
| Env Var        | Description                               | Default             |
|----------------|-------------------------------------------|---------------------|
| `MY_SERVER_KEY`| API key required in `Authorization` header| `data-loom-api-key` |
| `STORAGE_TYPE` | Storage backend (`badger` or `sqlite`)    | `badger`            |
| `STORAGE_PATH` | Path to data directory or DB file         | `./data`            |
| `PORT`         | WebSocket server port (planned)           | `8080`              |

## Running
```bash
go run ./server/cmd/data-loom-server/main.go
```

## Persistence Backends

Badger: Default backend. Embedded key-value store optimized for speed.
SQLite: Lightweight relational database backend. Created but not yet fully tested.
No persistence: Planned feature where the server can operate entirely in memory.