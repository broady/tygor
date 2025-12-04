# Devtools Example

A React + Vite application demonstrating tygor's vite plugin with full hot-reload across Go and TypeScript.

## Quick Start

```bash
bun install
bun dev
```

Open http://localhost:5173

## Scripts

- `npm run dev` - Start Go + Vite dev servers with hot-reload
- `npm run gen` - Regenerate TypeScript types from Go
- `npm run build` - Production build
- `npm run typecheck` - Type-check TypeScript

## How `bun dev` Works

This single command:
- Starts Go server with hot-reload (via `@tygor/vite-plugin`)
- Starts Vite dev server with HMR
- Vite proxies API requests to Go (no CORS needed)
- Editing Go types -> tygorgen runs -> TypeScript updates -> browser refreshes

## How It Works

The `@tygor/vite-plugin` handles everything:

```
Edit .go file
    |
Plugin detects change (chokidar)
    |
Runs prebuild (tygorgen)
    |
Starts new Go server on alternate port
    |
Health checks until ready
    |
Swaps proxy target (zero downtime)
    |
Kills old server
    |
Vite HMR picks up TypeScript changes
    |
Browser updates
```

Build errors show in a non-blocking overlay - the page remains interactive.

The Vite config auto-derives proxy routes from the generated manifest:

```javascript
import { registry } from "./src/rpc/manifest";

// Routes like /System/*, /Tasks/* proxy to Go
```

## Configuration

See `vite.config.js` for the plugin configuration:

```javascript
tygor({
  proxyPrefix: "/api",
  build: "go build -o ./.tygor/server .",
  buildOutput: "./.tygor/server",
  start: (port) => ({
    cmd: ["./.tygor/server"],
    env: { PORT: String(port) },
  }),
  rpcDir: "./src/rpc",
})
```
