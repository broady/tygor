# @tygor/vite-plugin

Vite plugin for [tygor](https://github.com/broady/tygor) with Go backend hot-reload and error overlay.

## Installation

```bash
npm install @tygor/vite-plugin
```

## Usage

Add the plugin to your Vite config:

```typescript
import { defineConfig } from "vite";
import { tygorDev } from "@tygor/vite-plugin";
import { registry } from "./src/rpc/manifest";

export default defineConfig({
  plugins: [
    tygorDev({
      build: "go build -o ./tmp/server .",
      start: (port) => ({
        cmd: `./tmp/server -addr=:${port}`,
      }),
      manifest: registry,
    }),
  ],
});
```

## Features

- **Hot reload**: Watches Go files and rebuilds/restarts the server on changes
- **Zero-downtime**: Starts new server on different port before swapping
- **Error overlay**: Shows build and runtime errors in-browser
- **Auto proxy**: Derives API routes from tygor manifest automatically
- **Devtools sidebar**: Expandable panel showing server status and RPC errors
- **Watchdog**: Automatically restarts crashed servers

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `start` | `(port) => { cmd, env?, cwd? }` | required | Function returning server start command |
| `build` | `string \| string[]` | - | Build command (e.g., `go build -o ./tmp/server .`) |
| `prebuild` | `string \| string[]` | - | Command to run before build (e.g., codegen) |
| `buildOutput` | `string` | - | Path to build output (creates parent directory) |
| `manifest` | `TygorRegistry` | - | Tygor manifest for auto-deriving proxy paths |
| `proxy` | `string[]` | - | Additional proxy path prefixes (e.g., `/health`, `/static`) |
| `watch` | `string[]` | `['**/*.go']` | Glob patterns to watch |
| `ignore` | `string[]` | `['node_modules', '.git', 'tmp', 'dist']` | Patterns to ignore |
| `health` | `string \| false` | `false` | Health check endpoint (false = TCP probe) |
| `port` | `number` | `8080` | Starting port to search from |
| `workdir` | `string` | `process.cwd()` | Working directory for Go commands |

## How It Works

```
Edit .go file
    ↓
Plugin detects change (chokidar)
    ↓
Runs prebuild (if configured)
    ↓
Runs build command
    ↓
Starts new server on fresh port
    ↓
Health checks pass
    ↓
Swaps traffic to new server
    ↓
Kills old server
```

The plugin maintains two server slots for zero-downtime reloads. If the new server fails to start, the old server continues serving requests.

## Error Handling

Build and runtime errors are displayed in an overlay injected into your app:

- **Build errors**: Syntax errors, type errors from `go build`
- **Runtime errors**: Server crashes, startup failures
- **RPC errors**: Failed API calls shown temporarily

The devtools UI uses Shadow DOM for style isolation.

## Proxy Behavior

When a `manifest` is provided, the plugin automatically proxies requests to your Go server based on service names:

```typescript
// If manifest contains paths like /Users/Get, /Tasks/List
// Plugin proxies all requests starting with /Users, /Tasks
```

Use `proxy` to add additional paths beyond those derived from the manifest:

```typescript
tygorDev({
  manifest: registry,
  proxy: ["/health", "/static", "/uploads"],
  // ...
})
```

## Documentation

For full documentation, see the [tygor repository](https://github.com/broady/tygor).

## License

MIT
