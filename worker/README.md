# Media Transform Worker

A Cloudflare Worker for media content transformation.

## Features

- M3U8 playlist processing with URL rewriting
- TS segment handling
- Token-based authentication
- URL encryption with MD5 signatures
- Automatic retry on 403 errors
- CORS support

## Prerequisites

- Node.js 18+
- A Cloudflare account
- Wrangler CLI (installed as dev dependency)

## Installation

```bash
cd worker
npm install
```

## Configuration

Edit `wrangler.toml` to customize settings:

```toml
name = "media-transform"
main = "dist/index.js"
compatibility_date = "2024-01-01"
minify = true

[vars]
UPSTREAM_API = "https://cbiz.yanhekt.cn"
VIDEO_HOST = "cvideo.yanhekt.cn"
MAGIC_KEY = "your-magic-key-here"
```

## Development

Run locally with hot reload (uses unobfuscated source):

```bash
npm run dev
```

This starts a local server at `http://localhost:8787`.

## Testing

```bash
npm test              # Run tests in watch mode
npm test -- --run     # Run tests once
npm run typecheck     # TypeScript type checking
```

## Build

Build obfuscated bundle for deployment:

```bash
npm run build
```

This creates `dist/index.js` with:
- Full minification via esbuild
- Code obfuscation (control flow flattening, dead code injection, string encoding)
- No readable keywords in output

## Deployment

### First-time Setup

1. Login to Cloudflare:
   ```bash
   npx wrangler login
   ```

2. Deploy the worker:
   ```bash
   npm run deploy
   ```

The worker will be deployed to `https://media-transform.<your-subdomain>.workers.dev`.

### Custom Domain (Optional)

Add to `wrangler.toml`:
```toml
routes = [
  { pattern = "api.yourdomain.com/*", zone_name = "yourdomain.com" }
]
```

## API Endpoints

### Health Check

```
GET /health
```

Returns `{"status":"ok"}`.

### Stream Endpoint

```
GET /stream?url={m3u8_url}&token={token}
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `url` | Yes | Original M3U8 URL (must be from allowed domain) |
| `token` | Yes | 32-character hex authentication token |

**Responses:**
- `200` - Processed M3U8 content
- `400` - Missing url parameter or invalid domain
- `403` - Missing or invalid token

### TS Endpoint

```
GET /ts/{filename}?base={base_url}&token={token}
```

| Parameter | Required | Description |
|-----------|----------|-------------|
| `filename` | Yes | TS segment filename (URL path) |
| `base` | Yes | Base URL for resolving paths |
| `token` | Yes | 32-character hex authentication token |

**Responses:**
- `200` - TS segment content
- `400` - Missing base parameter or invalid domain
- `403` - Missing or invalid token

## Security

- **Domain restriction**: Only `cvideo.yanhekt.cn` URLs are allowed
- **Token validation**: Tokens must be exactly 32 hexadecimal characters
- **No information disclosure**: Invalid tokens return generic 403 response

## Architecture

```
src/
├── index.ts              # Entry point and routing
├── md5.ts                # MD5 implementation
├── crypto.ts             # URL encryption and signing
├── token.ts              # Token fetching
├── fetch.ts              # HTTP client with retry
├── validation.ts         # Input validation
└── handlers/
    ├── health.ts         # GET /health
    ├── stream.ts         # GET /stream
    └── segment.ts        # GET /ts/*
```
