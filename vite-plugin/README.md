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
      workdir: "../server",  // Path to your Go module
      build: "go build -o ./tmp/server .",
      start: (port) => ({ cmd: `./tmp/server -port=${port}` }),
    }),
  ],
});
```

That's it. The plugin automatically:
- Runs `tygor gen` to generate TypeScript types (output to `rpcDir`)
- Starts a devtools server (`tygor dev`)
- Builds and runs your Go server
- Hot-reloads on file changes with zero downtime
- Shows errors in-browser

> **Note**: `workdir` sets the working directory for Go commands. In a typical monorepo with `client/` and `server/` directories, set this to the path of your Go module relative to `vite.config.ts`.

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
| `gen` | `boolean` | `true` | Run `tygor gen` automatically. Set to `false` to disable |
| `prebuild` | `string \| string[]` | - | Custom command to run after `tygor gen` but before build |
| `buildOutput` | `string` | - | Path to build output (creates parent directory) |
| `rpcDir` | `string` | `'./src/rpc'` | Output directory for `tygor gen` and discovery.json |
| `proxy` | `string[]` | - | Proxy paths (auto-derived from discovery.json if not set) |
| `watch` | `string[]` | `['**/*.go']` | Glob patterns to watch |
| `ignore` | `string[]` | `['node_modules', '.git', 'tmp', 'dist']` | Patterns to ignore |
| `health` | `string \| false` | `false` | Health check endpoint (false = TCP probe) |
| `port` | `number` | `8080` | Starting port to search from |
| `workdir` | `string` | `process.cwd()` | Working directory for Go commands and file watcher |

## Error Handling

Build and runtime errors are displayed in an overlay:

- **Build errors**: Syntax errors, type errors from `go build`
- **Runtime errors**: Server crashes, startup failures
- **RPC errors**: Failed API calls shown temporarily

The overlay uses Shadow DOM for style isolation.

## API Discovery

The plugin serves your API schema at `/__tygor/discovery`. The devtools sidebar uses this to display your services.

Discovery is enabled automatically - the plugin runs `tygor gen <rpcDir> --discovery` on startup and on each file change.

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

## Advanced: Custom Codegen

To run additional commands after `tygor gen` (e.g., custom codegen), use `prebuild`:

```typescript
tygorDev({
  prebuild: "go generate ./...",
  build: "go build -o ./tmp/server .",
  start: (port) => ({ cmd: `./tmp/server -port=${port}` }),
})
```

To disable `tygor gen` entirely (e.g., if you're using a different code generator), set `gen: false`:

```typescript
tygorDev({
  gen: false,
  prebuild: "my-custom-codegen ./src/rpc",
  build: "go build -o ./tmp/server .",
  start: (port) => ({ cmd: `./tmp/server -port=${port}` }),
})
```

## License

MIT
