For edge function version, please refer to [worker/README.md](worker/README.md)

# Video Proxy Server

A stateless video proxy server for HLS streaming with DRM protection. Designed for deployment on Linux VMs.

## Features

- **Path-based network mode**: `/external/` for CDN, `/intranet/` for internal IP mapping
- **Token caching**: 10-second in-memory cache per login token
- **Load balancing**: Round-robin, random, or first-available strategies
- **Retry logic**: Automatic retry with token refresh on 403 errors
- **Failed IP tracking**: 5-minute auto-recovery for failed intranet IPs
- **Config reload**: Via API or SIGHUP signal

## Quick Start

```bash
# Build
make build

# Run
./video-proxy

# Or with Docker
make docker-build
make docker-run
```

## API Endpoints

### Stream Endpoints

**External mode (via CDN):**
```
GET /external/stream?url=<m3u8_url>&token=<login_token>
GET /external/ts/<filename>?base=<base_url>&token=<login_token>
```

**Intranet mode (via IP mapping):**
```
GET /intranet/stream?url=<m3u8_url>&token=<login_token>
GET /intranet/ts/<filename>?base=<base_url>&token=<login_token>
```

### Management Endpoints

```
GET  /health                    - Health check
GET  /api/v1/config/mappings    - Get current IP mappings
POST /api/v1/config/reload      - Reload mappings from config file
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `UPSTREAM_API` | `https://cbiz.yanhekt.cn` | API base URL for token fetching |
| `VIDEO_HOST` | `cvideo.yanhekt.cn` | Video CDN hostname |
| `MAGIC_KEY` | (built-in) | Signature magic key |
| `LOG_LEVEL` | `info` | Logging level |
| `REQUEST_TIMEOUT` | `30s` | External request timeout |
| `INTRANET_TIMEOUT` | `8s` | Intranet request timeout |
| `MAPPINGS_FILE` | `./mappings.json` | Path to IP mappings config |

### Mappings Config File

`mappings.json` defines domain-to-IP mappings for intranet mode:

```json
{
  "cvideo.yanhekt.cn": {
    "type": "single",
    "ip": "10.0.34.24"
  },
  "cbiz.yanhekt.cn": {
    "type": "loadbalance",
    "ips": ["10.0.34.22", "10.0.34.21"],
    "strategy": "round_robin"
  }
}
```

**Mapping types:**
- `single`: Single IP mapping
- `loadbalance`: Multiple IPs with load balancing

**Strategies:**
- `round_robin`: Rotate through IPs sequentially
- `random`: Random IP selection
- `first_available`: Always use first available IP

### Config Reload

Reload mappings without restart:

```bash
# Via API
curl -X POST http://localhost:8080/api/v1/config/reload

# Via signal
kill -HUP <pid>
```

## Deployment

### Docker

```bash
# Build image
docker build -t video-proxy:latest .

# Run container
docker run -d \
  --name video-proxy \
  -p 8080:8080 \
  -v /path/to/mappings.json:/app/mappings.json \
  -e PORT=8080 \
  video-proxy:latest
```

### Systemd Service

Create `/etc/systemd/system/video-proxy.service`:

```ini
[Unit]
Description=Video Proxy Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/video-proxy
ExecStart=/opt/video-proxy/video-proxy
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
Environment=PORT=8080
Environment=MAPPINGS_FILE=/opt/video-proxy/mappings.json

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable video-proxy
sudo systemctl start video-proxy
```

## Development

```bash
# Install dependencies
make deps

# Run in development mode
make run-dev

# Format code
make fmt

# Run tests
make test
```

## Architecture

```
server/
├── cmd/proxy/main.go           # Entry point
├── internal/
│   ├── config/config.go        # Environment configuration
│   ├── crypto/crypto.go        # URL encryption & signatures
│   ├── handler/
│   │   ├── health.go           # Health check
│   │   ├── stream.go           # M3U8 stream proxy
│   │   ├── segment.go          # TS segment proxy
│   │   └── config.go           # Config API
│   ├── mapping/intranet.go     # IP mapping & load balancing
│   ├── proxy/client.go         # HTTP client with retry
│   └── token/token.go          # Video token cache
├── mappings.json               # Default IP mappings
├── Dockerfile
└── Makefile
```
