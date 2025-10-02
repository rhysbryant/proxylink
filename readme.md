# proxyLink

Simple forward web proxy that additionally supports tunnelling user traffic between the bridge and exit node,
encrypts and encapsulates traffic within web socket frames.

The connection between the bridge and the exit node can optionally use TLS. Let's Encrypt integration allows for automatic certificate generation.

## Modes of Operation

### 1. Standalone Mode
In this mode, the proxy acts as a direct HTTP proxy, forwarding requests to their destinations.

### 2. Bridge Mode
The proxy acts as an intermediary, forwarding encrypted WebSocket traffic to the next proxy in the chain.

### 3. Exit Node Mode
The proxy acts as the final node in the chain, decrypting WebSocket traffic and forwarding it to the destination.

## Flow Diagram

```plaintext
+----------+       +--------------+       +------------+       +--------+
|  User/OS | <---> | Bridge Proxy | <===> |  Exit Node | <---> | Target |
+----------+       +--------------+       +------------+       +--------+
```

example

Bridge proxy is configured as a proxy for the web browser

1. user request to https://example.com
2. request is sent to bridge
3. bridge connects to exit node and sends the request from the user
3. exit node connects to https://example.com

## Features

- **Modes**: Standalone, Bridge, Exit.
- **Encryption**: Optional WebSocket traffic encryption using a 32-byte key.
- **TLS Support**: Custom certificates or automatic Let's Encrypt integration.
- **Configurable**: Command-line flags for easy setup.
- **Logging**: Supports configurable log levels and formats (text or JSON).

## Usage

### Command-Line Flags

| Flag             | Description                                      |
|------------------|--------------------------------------------------|
| `--mode`         | Mode of operation: `standalone`, `bridge`, `exit`. |
| `--next`         | Address of the next proxy (required in bridge mode). |
| `--listen`       | Address to listen on (default: `:8080`).         |
| `--tls-cert`     | Path to TLS certificate file.                    |
| `--tls-key`      | Path to TLS key file.                            |
| `--ws-key`       | 32-byte key (in hex) for encrypting traffic.*    |
| `--lets-encrypt` | Enable Let's Encrypt support.                    |
| `--domain`       | Domain name for Let's Encrypt (required if enabled). |
| `--log-level`    | Logging level: `debug`, `info`, `warn`, `error`. |
| `--log-format`   | Log format: `text` (default) or `json`.          |

\* traffic from public addresses is blocked if no key is provided

### Example Commands

#### Standalone Mode
```bash
webproxy --mode standalone --listen :8080 --log-level info --log-format json
```

#### Bridge Mode
```bash
webproxy --mode bridge --next ws://next-proxy:8080 --ws-key <32-byte-hex-key>
```

#### Exit Node Mode
```bash
webproxy --mode exit --listen :8080 --ws-key <32-byte-hex-key>
```

#### Let's Encrypt
```bash
webproxy -mode exit --lets-encrypt -domain example.com --listen :443 --ws-key <32-byte-hex-key>
```

see config examples

#### Building
```bash
go build
```

## Service Support

WebProxy includes support for running as a system service using the [kardianos/service](https://github.com/kardianos/service) library. This allows you to install, start, stop, and uninstall the proxy as a background service on supported platforms (e.g., Windows, Linux, macOS).

### Service Commands

You can control the service using the `--service` flag. The following commands are supported:

| Command       | Description                              |
|---------------|------------------------------------------|
| `install`     | Installs the proxy as a system service.  |
| `uninstall`   | Uninstalls the proxy service.            |
| `start`       | Starts the installed proxy service.      |
| `stop`        | Stops the running proxy service.         |

### Example Usage

#### Install the Service
```bash
webproxy --service install
```

## License

This project is licensed under the GPLv3 License.