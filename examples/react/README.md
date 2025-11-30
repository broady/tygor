# React Example

A minimal React + Vite application demonstrating tygor's type-safe RPC client with full hot-reload across Go and TypeScript.

## Quick Start

```bash
bun install
bun dev
```

Open http://localhost:5173

This single command:
- Starts Go server with hot-reload (via [air](https://github.com/air-verse/air))
- Starts Vite dev server with HMR
- Vite proxies API requests to Go (no CORS needed)
- Editing Go types → tygorgen runs → TypeScript updates → browser refreshes

## How It Works

```
Edit .go file
    ↓
Air detects change
    ↓
tygorgen regenerates types (only if changed)
    ↓
Go server rebuilds & restarts
    ↓
Vite HMR picks up TypeScript changes
    ↓
Browser updates
```

The Vite config auto-derives proxy routes from the generated manifest:

```javascript
import { registry } from "./src/rpc/manifest";

// Routes like /System/*, /Tasks/* proxy to Go
```

## Code Snippets

### App Setup

<!-- [snippet:app-setup] -->
```go title="main.go"
// No CORS needed - Vite proxies API requests in dev, same-origin in prod
app := tygor.NewApp()

system := app.Service("System")
system.Register("Info", tygor.Query(GetRuntimeInfo))

tasks := app.Service("Tasks")
tasks.Register("List", tygor.Query(ListTasks))
tasks.Register("Create", tygor.Exec(CreateTask))
tasks.Register("Toggle", tygor.Exec(ToggleTask))

```
<!-- [/snippet:app-setup] -->

### Client Setup

<!-- [snippet:client-setup] -->
```tsx title="App.tsx"
import { useState, useEffect, FormEvent } from "react";
import { createClient } from "@tygor/client";
import { registry } from "./rpc/manifest";
import type { Task, RuntimeInfo } from "./rpc/types";

// No baseUrl needed - uses current origin (works with Vite proxy in dev, same-origin in prod)
const client = createClient(registry);

```
<!-- [/snippet:client-setup] -->

### React Component

<!-- [snippet:react-component] -->
```tsx title="App.tsx"
export default function App() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [newTask, setNewTask] = useState("");
  const [info, setInfo] = useState<RuntimeInfo | null>(null);

  const fetchTasks = async () => {
    setTasks(await client.Tasks.List({}));
  };

  useEffect(() => {
    fetchTasks();
  }, []);

  useEffect(() => {
    const fetchInfo = async () => {
      setInfo(await client.System.Info({}));
    };
    fetchInfo();
    const interval = setInterval(fetchInfo, 1000);
    return () => clearInterval(interval);
  }, []);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!newTask.trim()) return;
    await client.Tasks.Create({ title: newTask });
    setNewTask("");
    fetchTasks();
  };

  const handleToggle = async (id: number) => {
    await client.Tasks.Toggle({ id });
    fetchTasks();
  };

  const formatBytes = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  };

  return (
    <div>
      {info && (
        <div className="info">
          <strong>{info.version}</strong> | {info.num_goroutines} goroutines |{" "}
          {formatBytes(info.memory.alloc)} alloc | {info.memory.num_gc} GC
        </div>
      )}

      <h1>Tasks</h1>
      <form onSubmit={handleCreate}>
        <input
          value={newTask}
          onChange={(e) => setNewTask(e.target.value)}
          placeholder="New task..."
        />
        <button type="submit">Add</button>
      </form>
      <ul>
        {tasks.map((task) => (
          <li
            key={task.id}
            className={task.done ? "done" : ""}
            onClick={() => handleToggle(task.id)}
          >
            {task.done ? "✓" : "○"} {task.title}
          </li>
        ))}
      </ul>
    </div>
  );
}

```
<!-- [/snippet:react-component] -->
