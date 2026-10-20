# SecBan Repeater

SecBan Repeater is a ban replication daemon for the [SecBan framework](https://gitlab.com/secban). It listens on a UNIX domain socket for IPC notification messages sent by a local SecBan agent upon IP ban events and forwards (repeats) them to one or more remote SecBan agent instances over HTTP/HTTPS.

## Features

- IPC receiver via UNIX domain socket
- Ban event replication across multiple SecBan agents
- Support for HTTP and HTTPS endpoints
- Configurable HTTP request timeout per agent call

## Installation

Pre-compiled release packages are published in the GitLab package registry: https://gitlab.com/secban/secban-repeater/-/releases/permalink/latest

### Download and Extract

```sh
# Set target version, OS, and architecture
VERSION="v0.1.0" # replace with desired version
OS="linux"
ARCH="amd64" # or arm64

# Download archive from GitLab Generic Package Registry
curl -sSL -o "secban-repeater.tar.gz" \
  "https://gitlab.com/api/v4/projects/secban%2Fsecban-repeater/packages/generic/secban-repeater/${VERSION}/secban-repeater-${OS}-${ARCH}-${VERSION}.tar.gz"

# Extract archive
mkdir -p /tmp/secban-install
tar -xzf "secban-repeater.tar.gz" -C /tmp/secban-install
cd /tmp/secban-install
```

### Install

Run the included install script as `root`:

```sh
sudo ./install.sh
```

The script performs the following steps:
1. Creates `secban` system group and `secban` system user.
2. Installs daemon `/usr/sbin/secban-repeater`.
3. Creates runtime directory `/var/run/secban`.
4. Copies default configuration to `/etc/secban/repeater.yaml` (if not already present).
5. Installs and enables system service (`systemd` or `OpenRC`).

### Start Service

```sh
# Systemd
sudo systemctl start secban-repeater.service
sudo systemctl status secban-repeater.service

# OpenRC
sudo rc-service secban-repeater start
sudo rc-service secban-repeater status
```

## Configuration

Main configuration file is located at `/etc/secban/repeater.yaml`.

### Configuration Options

| Option | Type | Description |
|---|---|---|
| `socket_path` | string | UNIX domain socket path to listen on. Default: `/var/run/secban/repeater.sock`. |
| `timeout` | string | HTTP request timeout duration per target agent (e.g. `5s`, `10s`). Default: `5s`. |
| `agents` | list of strings | List of target SecBan agent endpoints to repeat bans to (HTTP/HTTPS URLs). |
| `log_level` | string | Logging level (`debug`, `info`, `warn`, `error`). Default: `info`. |

## Examples

### 1. Standard Configuration (`/etc/secban/repeater.yaml`)

```yaml
# SecBan Repeater Configuration

# Log level: debug, info, warn, error
log_level: info

# UNIX socket path where secban-repeater listens for IPC events from the local agent
socket_path: /var/run/secban/repeater.sock

# HTTP request timeout per agent call
timeout: 5s

# List of remote or local secban-agent endpoints to repeat bans to.
# Supports HTTP/HTTPS (e.g. http://10.0.0.2:8080)
agents:
  - http://10.0.0.2:1024
  - http://10.0.0.3:1024
```

### 2. SecBan Agent Integration

To forward ban events from `secban-agent` to `secban-repeater`, add the repeater socket path to rule definitions in `/etc/secban/agent.yaml` or rule files under `/etc/secban/rules.d/`:

```yaml
rules_default:
  sockets:
    - /var/run/secban/repeater.sock
```

### 3. Command-Line Options

`secban-repeater` accepts the following command-line flags:

| Flag | Default | Description |
|---|---|---|
| `-config` | `/etc/secban` | Path to directory containing `repeater.yaml`. |
