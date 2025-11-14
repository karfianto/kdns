![Alt text](image-1.png)
# kdns

kdns is a local DNS server and traffic inspection tool designed to help you monitor and analyze DNS traffic on your localhost. This tool provides a REST API for querying and modifying DNS configuration information and a real-time web interface for visualizing DNS logs.

## Motivation

Inspecting DNS traffic can be essential for understanding how your applications resolve domain names and identifying any potential issues. kdns aims to simplify the process of modifying DNS records, monitoring DNS queries and responses on your local system.

## Features

- External DNS Resolver support
- DNS Record Customization 
- REST API for querying and chaning DNS configuration
- Real-time web interface for visualizing DNS logs
- Fast, simple, and easy-to-use 

### New: Redpanda/Kafka Streaming

- Publish DNS events to a Redpanda topic
- Optionally consume the same topic and forward to the WebSocket UI
- Decoupled logging: applications can subscribe to the topic for real-time DNS events

## REST API Documentation

The REST API provided by kdns allows you to retrieve DNS information programmatically. Here are some of the available endpoints:

- **GET http://localhost:8080/api/config**: Retrieve current kdns configuration.
- **POST http://localhost:8080/api/config**: Update kdns configuration.


## Usage

kdns offers a real-time web interface for visualizing DNS logs. To view the web interface, follow these steps:

1. Clone this repository:

   ```shell
   git clone https://github.com/karfianto/kdns.git
   cd kdns
   ```

2. Build the package
 
    ```shell
    go build
    ```
    Or download precompiled binaries in [release page](https://github.com/karfianto/kdns/releases).

3. Run the binary

    For Windows
    ```shell
    kdns.exe
    ```
    For Linux
    ```shell
    ./kdns_linux64
    ```
    For Mac
    ```shell
    ./kdns_macos
    ```
    ![Alt text](image-2.png)

4. Open your host Network Adapter Setting and change your DNS Server to `127.0.0.1`

    ![Alt text](image.png)

5. Open your web browser and visit `http://localhost:8080` to access the real-time web interface.

    ![Alt text](image-3.png)

## Environment Variables

Before running kdns, you can configure its behavior using environment variables stored in a `.env` file. Here are the available environment variables and their descriptions:

- `HTTP_BIND_URL`: The IP address to bind to. Set it to `0.0.0.0` to listen on all available network interfaces.

- `HTTP_PORT`: The port number to listen on for incoming HTTP requests. Default is `8080`.

- `HTTP_MODE`: Set the mode for your application. You can use `production` for production deployments and `development` for local development and debugging.

- `DNS_DEBUG`: Set this to `true` to enable DNS debugging in terminal. 

### Redpanda/Kafka Integration (Optional)

Enable these to publish/consume DNS events via Redpanda (Kafka API):

- `REDPANDA_BROKERS`: Comma-separated broker addresses, e.g. `localhost:9092`
- `REDPANDA_TOPIC`: Topic name to publish/consume, e.g. `kdns.dns.events`
- `REDPANDA_ENABLE_PUBLISH`: Set to `true` to publish DNS request/response events
- `REDPANDA_ENABLE_CONSUME`: Set to `true` to consume the topic and forward received messages to connected WebSocket clients
- `REDPANDA_CONSUMER_GROUP`: (optional) Consumer group ID for the web process. Defaults to `kdns-web` if not set

### Example .env File

Here's an example of a `.env` file that you can customize for your needs:

```plaintext
HTTP_BIND_URL=0.0.0.0
HTTP_PORT=8080
HTTP_MODE=production
DNS_DEBUG=true

# Redpanda streaming (optional)
REDPANDA_BROKERS=localhost:9092
REDPANDA_TOPIC=kdns.dns.events
REDPANDA_ENABLE_PUBLISH=true
REDPANDA_ENABLE_CONSUME=true
REDPANDA_CONSUMER_GROUP=kdns-web
```

## Redpanda Streaming Overview

When enabled, kdns will:

- Publish a JSON event for every DNS request and response processed. Example payloads:

  Request event
  ```json
  {
    "timestamp": "2025-01-01T12:34:56.789Z",
    "event_type": "request",
    "source_ip": "127.0.0.1",
    "query_name": "example.com.",
    "query_type": "A",
    "server_addr": ":53"
  }
  ```

  Response event
  ```json
  {
    "timestamp": "2025-01-01T12:34:56.800Z",
    "event_type": "response",
    "source_ip": "127.0.0.1",
    "answers": [
      "example.com.  300 IN A 93.184.216.34"
    ],
    "server_addr": ":53"
  }
  ```

- If `REDPANDA_ENABLE_CONSUME=true`, read from the same topic and forward raw messages to the existing WebSocket endpoint (`/ws`). This keeps the UI in sync with the topic and allows other consumers to process the same stream independently.

Notes:
- If Redpanda env vars are not fully set, kdns will skip streaming and continue serving the WebSocket-only logs as before.
- The Web UI still receives human-readable log lines directly from the server even when publishing is enabled; consuming from the topic will additionally forward JSON messages to clients.

## How to Contribute
Contributions to kdns are welcome and encouraged! Here's how you can contribute:

1. Fork the repository to your GitHub account.
2. Create a new branch for your feature or bug fix:
    ```shell
    git checkout -b feature/my-feature
    ```
3. Make your changes and commit them with clear commit messages.
4. Push your changes to your fork:
    ```shell
    git push origin feature/my-feature
    ```
5. Create a pull request (PR) against the main branch of this repository.
