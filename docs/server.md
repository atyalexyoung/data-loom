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
| `MY_SERVER_KEY`| API key required in `Authorization` header. If not set or blank, the server will not check for an `Authorization` header and accept all incoming connection requests (If Client ID is valid)  | `""`  |
| `STORAGE_TYPE` | Storage backend (`badger`, `sqlite`, `none`, or `""`)    | `""` |
| `STORAGE_PATH` | Path to data directory or DB file         | `./tmp/data/`       |
| `PORT_NUMBER`  | WebSocket server port                     | `8080`              |

## Running
```bash
go run ./server/cmd/data-loom-server/main.go
```

## Persistence Backends

Badger: Default backend. Embedded key-value store optimized for speed.
SQLite: Lightweight relational database backend. Created but not yet fully tested.
No persistence: Server operates entirely in memory and doesn't persist any data.