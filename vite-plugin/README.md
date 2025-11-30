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

export default defineConfig({
  plugins: [
    tygorDev({
      prebuild: "go run . -gen -out ./client/src/rpc",
      build: "go build -o ./tmp/server .",
      start: (port) => ({
        cmd: `./tmp/server -addr=:${port}`,
      }),
    }),
  ],
});
```

## Features

- **Hot reload**: Watches Go files and rebuilds/restarts the server on changes
- **Zero-downtime**: Starts new server on different port before swapping
- **Error overlay**: Shows build and runtime errors in-browser
- **Auto proxy**: Derives API routes from discovery.json automatically
- **Devtools sidebar**: Expandable panel showing server status and RPC errors
- **Watchdog**: Automatically restarts crashed servers

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `start` | `(port) => { cmd, env?, cwd? }` | required | Function returning server start command |
| `build` | `string \| string[]` | - | Build command (e.g., `go build -o ./tmp/server .`) |
| `prebuild` | `string \| string[]` | - | Command to run before build (e.g., codegen) |
| `buildOutput` | `string` | - | Path to build output (creates parent directory) |
| `rpcDir` | `string` | `'./src/rpc'` | Directory containing generated RPC files (for discovery.json and proxy paths) |
| `proxy` | `string[]` | - | Proxy path prefixes (auto-derived from discovery.json if not specified) |
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

## API Discovery

The plugin serves your API schema at `/__tygor/discovery` for runtime introspection. This enables devtools and API browsers to display your services and types.

**Setup:**

1. Enable discovery in your Go app's code generation:
```go
tygorgen.FromApp(app).
    WithDiscovery().  // Generates discovery.json
    ToDir("./client/src/rpc")
```

2. The plugin reads `discovery.json` from `rpcDir` (default: `./src/rpc`) and serves it at `/__tygor/discovery`.

3. Fetch the schema in your client:
```typescript
const schema = await fetch("/__tygor/discovery").then(r => r.json());
// schema.Services - array of services with endpoints
// schema.Types - array of type definitions
```

The devtools sidebar automatically uses this endpoint to display registered services.

## Proxy Behavior

The plugin automatically reads `discovery.json` from `rpcDir` and proxies requests based on service names:

```typescript
// If discovery.json contains services like "Users", "Tasks"
// Plugin proxies all requests starting with /Users, /Tasks
```

Use `proxy` to override auto-detection or add paths not in your schema:

```typescript
tygorDev({
  proxy: ["/health", "/static", "/uploads"],
  // ...
})
```

## Documentation

For full documentation, see the [tygor repository](https://github.com/broady/tygor).

## License

MIT
