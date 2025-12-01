# @tygor/vite-plugin

Vite plugin for [tygor](https://github.com/broady/tygor) with Go backend hot-reload, error overlay, and devtools.

## Installation

```bash
npm install @tygor/vite-plugin
```

## Quick Start

```typescript title="vite.config.ts"
import { defineConfig } from "vite";
import { tygorDev } from "@tygor/vite-plugin";

export default defineConfig({
  plugins: [
    tygorDev({
      build: "go build -o ./tmp/server .",
      start: (port) => ({ cmd: `./tmp/server -port=${port}` }),
    }),
  ],
});
```

That's it. The plugin automatically:
- Runs `tygor gen` to generate TypeScript types
- Starts a devtools server (`tygor dev`)
- Builds and runs your Go server
- Hot-reloads on file changes with zero downtime
- Shows errors in-browser

## How It Works

```
┌─────────────────────────────────────────────┐
│ Vite Dev Server (:5173)                     │
│  ├── /__tygor/*  → tygor dev (devtools)     │
│  └── /Service/*  → your Go server           │
└─────────────────────────────────────────────┘
```

The plugin manages two processes:
1. **tygor dev** - Stays up always, provides devtools and discovery
2. **Your Go server** - Restarts on code changes

When you edit a `.go` file:
1. Plugin detects change
2. Runs `tygor gen` to update types
3. Builds your server
4. Starts new server on fresh port
5. Health checks pass
6. Swaps traffic to new server
7. Kills old server

**Zero downtime**: The old server keeps handling requests until the new one is ready. Your frontend state persists.

## Features

- **Hot reload**: Watches Go files and rebuilds/restarts the server
- **Zero-downtime**: New server starts before old one stops
- **Error overlay**: Build and runtime errors shown in-browser
- **Auto proxy**: Derives API routes from discovery.json
- **Devtools sidebar**: Shows server status, services, and RPC errors
- **Watchdog**: Automatically restarts crashed servers

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `start` | `(port) => { cmd, env?, cwd? }` | required | Function returning server start command |
| `build` | `string \| string[]` | - | Build command (e.g., `go build -o ./tmp/server .`) |
| `prebuild` | `string \| string[]` | - | Custom command before build (overrides `tygor gen`) |
| `buildOutput` | `string` | - | Path to build output (creates parent directory) |
| `rpcDir` | `string` | `'./src/rpc'` | Directory for generated files and discovery.json |
| `proxy` | `string[]` | - | Proxy paths (auto-derived from discovery.json if not set) |
| `watch` | `string[]` | `['**/*.go']` | Glob patterns to watch |
| `ignore` | `string[]` | `['node_modules', '.git', 'tmp', 'dist']` | Patterns to ignore |
| `health` | `string \| false` | `false` | Health check endpoint (false = TCP probe) |
| `port` | `number` | `8080` | Starting port to search from |
| `workdir` | `string` | `process.cwd()` | Working directory for Go commands |

## Error Handling

Build and runtime errors are displayed in an overlay:

- **Build errors**: Syntax errors, type errors from `go build`
- **Runtime errors**: Server crashes, startup failures
- **RPC errors**: Failed API calls shown temporarily

The overlay uses Shadow DOM for style isolation.

## API Discovery

The plugin serves your API schema at `/__tygor/discovery`. The devtools sidebar uses this to display your services.

To enable discovery, run `tygor gen` with `--discovery`:

```bash
tygor gen ./src/rpc --discovery
```

Or set a custom `prebuild` that includes the flag.

## Proxy Behavior

The plugin reads `discovery.json` and proxies requests based on service names:

```typescript
// If discovery.json contains services "Users", "Tasks"
// Plugin proxies: /Users/* → Go server, /Tasks/* → Go server
```

Override with `proxy` option:

```typescript
tygorDev({
  proxy: ["/health", "/static", "/uploads"],
  // ...
})
```

## Advanced: Custom Prebuild

To override the default `tygor gen`, use `prebuild`:

```typescript
tygorDev({
  prebuild: "go run ./cmd/gen -out ./src/rpc",
  build: "go build -o ./tmp/server .",
  start: (port) => ({ cmd: `./tmp/server -port=${port}` }),
})
```

## License

MIT
