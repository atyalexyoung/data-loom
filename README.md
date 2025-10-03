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

Currently there is only [Badger](https://github.com/hypermodeinc/badger?tab=readme-ov-file#badgerdb) implementation for persistent storage with SQLite being impleemented but untested yet.

## Current State
- Core functionality implemented: topic creation, subscription, publishing, persistence.
- Some simple tests in place for message sending and receiving, along with simple stress tests, but not very robust yet.
- Not yet fully hardened for production use.
- Super basic auth with api key that is passed with initial Authorization header.
- **Server binaries** for Linux, Windows, and macOS are available in the [GitHub Releases](https://github.com/atyalexyoung/data-loom/releases).
- **C# SDK** is published as a [NuGet package](https://www.nuget.org/packages/DataLoom.CSharp/) and can be installed directly in projects.

## Upcoming
- C# SDK is created with some testing + stress testing
- SQLite persistence is created but untested.
- Server pre-built binary for use without compilation and maybe a docker image.
- Better and more robust testing.
- Stretch goal of a Typescript SDK
- Configurable IP Address Binding
- Fixing persistence logging and ACKs when "no-persistence" is configured

## Stress Test Results

To evaluate the performance of Data Loom under load, I ran stress tests with increasing numbers of clients and messages per second. The tests measure message latency, publish time, and the ability of the server to handle concurrent clients. I don't think the tests were thorough or robust enough to tell definitively exactly how the server will perform under various loads. There are also many factors in the testing environment that could effect the results of the test.

### Test Summary

- **Clients tested:** 10 → 1000  
- **Message rates:** up to 1,000,000 msgs/sec  
- **Metrics captured:** Failures, Messages Received, Average/Min/Max Latency (ms), Publish Time (ms)  

### Results Charts

#### Number of Clients vs. Latency

Below is a chart showing **Number of Clients vs Latency**. As the client count increases, the latency rises, highlighting how the server handles many clients.

![Publish Time vs Clients](/images/Clients.png)

You can see how up to 200 clients has a fairly stable max latency, and afterwards it starts to rise. 

> Note: More thorough testing needs to be done to get a better understanding of how the server performs under load. There are many factors that go into this, such as the computers running both the server and the client, the network they are connected over, and other things.


#### Messages per Second vs. Latency

Below is a chart showing **Frequency of messages vs Latency**. The client count was set at 20, and as the frequency of messages being sent from all clients increases, the and latency rises, highlighting how the server handles heavier message loads. This means that when the test was configured with 500 messages per second, it is 20 clients **all** sending 500 messages per second, so effectively this would be 10000 messages per second at full load.

![Publish Time vs Clients](/images/frequency.png)

You can see how up to about 600 messages per second from 20 clients has a fairly stable max latency, and afterwards it starts to rise. 

> Note: More thorough testing needs to be done to get a better understanding of how the server performs under load. There are many factors that go into this, such as the computers running both the server and the client, the network they are connected over, and other things.

### Sample Data

The clients data:
```csv
Timestamp,Clients,Failures,MessagesReceived,AvgLatencyMs,MinLatencyMs,MaxLatencyMs,PublishTimeMs
2025-10-02T22:28:13.4893269Z,75,0,5625,925.51,2,2333,4838
2025-10-02T22:32:05.2547624Z,10,0,100,136.24,34,335,354
2025-10-02T22:32:28.3471200Z,25,0,625,402.80,39,904,967
2025-10-02T22:32:50.6655127Z,50,0,2500,1047.18,6,2116,2277
2025-10-02T22:33:21.8879930Z,100,0,10000,1248.48,1,3400,7914
2025-10-02T22:34:09.4160002Z,150,0,22500,1390.47,1,3351,12573
2025-10-02T22:34:52.2522279Z,200,0,40000,1437.45,1,3750,18539
2025-10-02T22:35:49.0423092Z,300,28,90000,2431.64,0,11926,33543
2025-10-02T22:37:00.1784610Z,500,83,250000,2704.46,0,18210,42397
2025-10-02T22:39:28.1855123Z,750,33,562500,2020.50,1,11231,60607
2025-10-02T22:41:36.7584084Z,1000,25,1000000,2026.35,0,11680,89235
```

The frequency data:
```csv
Timestamp,Clients,Topics,MsgPerSec,DurationSec,Failures,MessagesReceived,AvgLatencyMs,MinLatencyMs,MaxLatencyMs
2025-10-03T00:57:18.2422707Z,20,4,50,30,0,101379,424.47,0,3335
2025-10-03T00:59:02.7097135Z,20,4,100,30,0,111813,352.07,0,4600
2025-10-03T01:00:40.1280913Z,20,4,250,30,0,139713,358.46,0,2985
2025-10-03T01:02:29.7320704Z,20,4,500,30,0,134635,361.97,0,2355
2025-10-03T01:04:07.3818972Z,20,4,1000,30,0,17429,5660.73,1,31087
2025-10-03T01:05:56.6544754Z,20,4,750,30,0,18584,4111.55,1,30348
2025-10-03T01:09:54.4747670Z,20,4,1250,30,0,2868,4842.58,2201,8298
2025-10-03T01:15:55.2940682Z,20,4,600,30,0,125489,435.74,0,3969
2025-10-03T01:50:54.9344854Z,20,4,750,30,0,64353,1910.73,0,16450
2025-10-03T01:54:09.6408795Z,20,4,700,30,0,23913,6799.71,1,35240
```

>These results are part of ongoing testing and are intended for learning purposes. The metrics may vary based on the server environment, client hardware, and network conditions.


## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

This project uses [Badger](https://github.com/hypermodeinc/badger?tab=readme-ov-file#badgerdb), licensed under the Apache License 2.0.
