# Getting Started

NOTE: These getting started notes are applicable now, but in future work the DataLoom Go server is planned to be a pre-built binary, and all SDKs will be packages with whatever the associated language's package management system is (e.g. NuGet for C#, NPM for TypeScript).

## Prerequisites
- Go 1.20+ installed
- .NET 8 SDK installed (if using the C# client)
- Docker (optional, if using the Docker container)

## Running the Go Server
1. Clone the repo:
2. build the go project at /server/cmd/data-loom-server/main.go
3. Set environment variables for configuration:

    - MY_SERVER_KEY for the API key that you want clients to send as the Authorization header in intial connect message to server. The default is no API Key required and the server will accept all connections.

    - STORAGE_TYPE for the type of underlying storage to use. The current options are:
        - badger
        - sqlite
        - none (same as default for no-persistence)
        - "" (empty string, same as default for no-persistence)
        
        The default is no-persistence and nothing will be stored
        .
    - STORAGE_PATH sets the path to location of where the persistent storage actually keeps its files (e.g. .db file for sqlite). The default is under "/tmp/data/"
    
    - PORT_NUMBER sets the port number that the server will serve on. The default is 8080.
4. Run the server and it will be on port 8080 by default. Configuration for this is planned.

## Using C# Client
NOTE: This is planned to be a NuGet package, but isn't currently.
1. Build the SDK at /sdks/csharp/DataLoom.SDK/
2. See the [C# SDK documentation](/docs/csharp-client.md) file for how to use the C# specific SDK.


## Raw Web Socket Connection
If there isn't an SDK for a particular langauge you want to use, then you can still use the service by manually connecting to the server. 

The server uses websocket connection at /ws on port 8080 by default (port configuraion is planned to be implemented.) See the [API Documentation](/docs/api.md) for how to structure messages and connect to the server manually.

## Quick Setup / Test
This section will be a guide to quickly get set up and test and is planned to be worked out soon.