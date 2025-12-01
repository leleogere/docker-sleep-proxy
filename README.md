# Docker Sleep Proxy

[![Docker Image CI](https://github.com/bvidotto/docker-sleep-proxy/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/bvidotto/docker-sleep-proxy/actions/workflows/docker-publish.yml)
[![GitHub release](https://img.shields.io/github/release/bvidotto/docker-sleep-proxy.svg)](https://github.com/bvidotto/docker-sleep-proxy/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight reverse proxy for Docker containers that automatically manages container lifecycle based on traffic. Perfect for resource-constrained environments like Raspberry Pi where you want to run multiple services but conserve memory.

## Installation

Pull the latest image from GitHub Container Registry:

```bash
docker pull ghcr.io/bvidotto/docker-sleep-proxy:latest
```

**Supported Platforms:**
- `linux/amd64` - x86_64 / AMD64
- `linux/arm64` - ARM 64-bit (Raspberry Pi 4, Apple Silicon)
- `linux/arm/v7` - ARM 32-bit (Raspberry Pi 3)

## Features

- 🔄 **Auto-start on traffic** - Containers start automatically when accessed
- 💤 **Auto-sleep on inactivity** - Containers stop after configurable idle time
- 📊 **Loading page** - Shows a beautiful loading screen while containers wake up
- 🏥 **Health checks** - Waits for containers to be fully ready before proxying
- 🎯 **Minimal footprint** - Only ~2.7 MiB of memory usage
- 🛑 **Manual shutdown** - REST endpoint to stop containers on demand
- 🔧 **Configurable** - All settings via environment variables

## How It Works

1. When traffic arrives, the proxy checks if target containers are running
2. If stopped, it starts them and shows a loading page
3. Once containers pass health checks, traffic is proxied through
4. After configured inactivity period, containers are automatically stopped
5. The proxy itself stays running, using minimal resources

## Quick Start

### Building from Source

```yaml
version: '3.8'

services:
  sleep-proxy:
    build: ./proxy
    ports:
      - '8000:8000'
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TARGET_SERVICE=myapp
      - TARGET_PORT=8080
      - SLEEP_TIMEOUT=3600  # 1 hour
    networks:
      - app-network
    restart: unless-stopped

  myapp:
    image: your-app-image
    expose:
      - '8080'
    networks:
      - app-network

networks:
  app-network:
```

### Run

```bash
docker compose up -d
```

Access your app at `http://localhost:8000`

## Configuration

All configuration is done via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TARGET_SERVICE` | ✅ Yes | - | Name of the Docker service to proxy to |
| `TARGET_SERVICE_DISPLAY_NAME` | No | `TARGET_SERVICE` | Friendly name to display on loading page |
| `TARGET_SERVICE_ICON` | No | - | Icon URL or `sh:appname` (see below) |
| `TARGET_PORT` | ✅ Yes | - | Port of the target service |
| `PROXY_PORT` | No | `8000` | Port the proxy listens on |
| `SLEEP_TIMEOUT` | No | `86400` | Seconds of inactivity before stopping containers (24h default) |
| `CHECK_INTERVAL` | No | `5` | Seconds between health checks during startup |
| `ENDPOINT_PREFIX` | No | `sleep-proxy` | Prefix for proxy management endpoints |
| `EXCLUSION_LABEL` | No | `sleep-proxy.exclude` | Label to exclude containers from lifecycle management |
| `DOCKER_HOST` | No | - | Docker host URL (e.g., `tcp://remote-docker:2375` for remote Docker or through proxy) |

## Customizing Loading Page
 
 You can customize the loading page with a friendly name and an icon.
 
 ### Service Name
 Set `TARGET_SERVICE_DISPLAY_NAME` to change the title from "Starting [TARGET_SERVICE]" to "Starting [Your Name]".
 
 ### Service Icon
 Set `TARGET_SERVICE_ICON` to display an icon instead of the default spinner.
 
 **Supported formats:**
 - **Direct URL**: `https://example.com/icon.png`
 - **Selfh.st Icons**: Use `sh:` prefix to fetch from [selfh.st/icons](https://selfh.st/icons)
   - `sh:bentopdf` -> Fetches Bentopdf SVG icon
   - `sh:bentopdf.png` -> Fetches Bentopdf PNG icon
   - `sh:bentopdf.webp` -> Fetches Bentopdf WebP icon
 
 ## Management Endpoints

The proxy exposes management endpoints at `/<ENDPOINT_PREFIX>/`:

### Health Check
```bash
curl http://localhost:8000/sleep-proxy/health
```

Returns:
- `{"status":"ready"}` - Containers are running and ready
- `{"status":"starting"}` - Containers are starting up

### Manual Shutdown
```bash
curl http://localhost:8000/sleep-proxy/shutdown
```

Or simply visit in browser: `http://localhost:8000/sleep-proxy/shutdown`

Returns: `{"status":"success","message":"Containers stopped"}`

## Excluding Containers

You can exclude specific containers from the sleep-proxy lifecycle management by adding a label:

```yaml
services:
  always-on-db:
    image: postgres
    labels:
      - "sleep-proxy.exclude=true"  # This container won't be stopped
    networks:
      - app-network

  myapp:
    image: your-app
    # This container will be managed (started/stopped) by the proxy
    networks:
      - app-network
```

The label name is configurable via the `EXCLUSION_LABEL` environment variable (defaults to `sleep-proxy.exclude`). Any container with this label will:
- Never be stopped during auto-sleep
- Never be stopped by the manual shutdown endpoint
- Continue running independently

This is useful for:
- **Databases** - Keep database containers always running
- **Cache services** - Redis, Memcached that should stay warm
- **Background workers** - Long-running tasks that shouldn't be interrupted
- **Monitoring tools** - Keep observability services active

## Health Check Support

The proxy supports two methods for determining container readiness:

1. **Docker Health Checks** (preferred) - Uses container's native healthcheck
2. **HTTP Fallback** - Makes HTTP request to target and waits for 200 OK

### Example with Docker Health Check

```yaml
myapp:
  image: your-app
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 5s
    timeout: 10s
    retries: 3
```

## Memory Usage

Measured on a typical setup:

- **sleep-proxy**: ~2.7 MiB (always running)
- **Target containers**: 0 MiB when sleeping, normal usage when active

Example with Stirling PDF:
- Active: ~209 MiB
- Sleeping: 0 MiB
- **Savings**: 209 MiB per inactive service

## Complete Example

```yaml
services:
  sleep-proxy:
    build: ./proxy
    ports:
      - '8000:8000'
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - PROXY_PORT=8000
      - TARGET_SERVICE=stirling-pdf
      - TARGET_PORT=8080
      - SLEEP_TIMEOUT=3600      # 1 hour
      - CHECK_INTERVAL=5        # 5 seconds
      - ENDPOINT_PREFIX=admin   # Custom prefix
    networks:
      - app-network
    restart: unless-stopped

  stirling-pdf:
    image: docker.stirlingpdf.com/stirlingtools/stirling-pdf:latest-ultra-lite
    expose:
      - '8080'
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:8080/api/v1/info/status"]
      interval: 5s
      timeout: 10s
      retries: 10
    networks:
      - app-network

  redis:
    image: redis:alpine
    labels:
      - "sleep-proxy.exclude=true"  # Keep Redis always running
    networks:
      - app-network

networks:
  app-network:
```

## Project Structure

```
.
├── proxy/
│   ├── main.go           # Entry point and SleepProxy struct
│   ├── config.go         # Configuration loading
│   ├── docker.go         # Container lifecycle management
│   ├── health.go         # Health check logic
│   ├── handlers.go       # HTTP handlers
│   ├── static/
│   │   ├── loading.html  # Loading page template
│   │   ├── loading.css   # Styles
│   │   └── loading.js    # Health polling script
│   ├── Dockerfile        # Multi-stage build
│   ├── go.mod
│   └── go.sum
└── compose.yml           # Docker Compose configuration
```

## How It Works Internally

1. **Traffic Detection**: Every request updates the activity timestamp
2. **Container Management**: Monitors all containers in the same Docker Compose project
3. **Self-Exclusion**: Proxy automatically excludes itself from start/stop operations
4. **Background Monitor**: Checks activity every 10 seconds and triggers sleep when threshold exceeded
5. **Loading Page**: Served while waiting for health checks to pass
6. **Reverse Proxy**: Standard HTTP reverse proxy once containers are ready

## Remote Docker / Docker Proxy

You can use the sleep-proxy with a remote Docker daemon or through a Docker proxy by setting the `DOCKER_HOST` environment variable:

```yaml
services:
  sleep-proxy:
    build: ./proxy
    ports:
      - '8000:8000'
    environment:
      - DOCKER_HOST=tcp://remote-docker.local:2375
      - TARGET_SERVICE=myapp
      - TARGET_PORT=8080
    # No need to mount docker.sock when using remote Docker
    networks:
      - app-network
    restart: unless-stopped
```

**Use cases:**
- Managing containers on a remote Docker host
- Using through docker-socket-proxy for security
- Connecting to Docker over TLS
- Running proxy outside of Docker (as standalone binary)

**Example with docker-socket-proxy:**

```yaml
services:
  docker-proxy:
    image: tecnativa/docker-socket-proxy
    environment:
      - CONTAINERS=1
      - POST=1
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    networks:
      - docker-api

  sleep-proxy:
    build: ./proxy
    environment:
      - DOCKER_HOST=tcp://docker-proxy:2375
      - TARGET_SERVICE=myapp
      - TARGET_PORT=8080
    networks:
      - docker-api
      - app-network
```

## Development

### Build Locally

```bash
cd proxy
docker build -t sleep-proxy .
```

### Run Tests

```bash
# Start with short timeout for testing
SLEEP_TIMEOUT=60 docker compose up -d

# Access the app
curl http://localhost:8000

# Wait 60+ seconds, containers should auto-sleep

# Check status
docker compose ps
```

## Requirements

- Docker Engine with Docker Compose
- Access to Docker socket (`/var/run/docker.sock`)
- Target containers must be in the same Docker Compose project

## Limitations

- Only works with HTTP traffic (no TCP/UDP proxying)
- Target containers must be in the same Docker Compose project
- Requires Docker socket access (security consideration)
- All containers in the project are managed together (not individually)

## Use Cases

- **Home Lab**: Run multiple services on Raspberry Pi without overwhelming RAM
- **Development**: Auto-sleep unused dev environments
- **Cost Savings**: Reduce cloud resource usage for low-traffic apps
- **Energy Efficiency**: Minimize power consumption for rarely-used services

