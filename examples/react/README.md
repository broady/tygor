# React Example

A minimal React + Vite application demonstrating tygor's type-safe RPC client.

## Quick Start

```bash
# Terminal 1: Start Go server
make run

# Terminal 2: Start Vite dev server
make dev
```

Open http://localhost:5173

## Code Snippets

### App Setup

<!-- [snippet:app-setup] -->
```go title="main.go"
app := tygor.NewApp().
	WithMiddleware(middleware.CORS(middleware.CORSAllowAll)).
	WithUnaryInterceptor(middleware.LoggingInterceptor(nil))

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

const client = createClient(registry, {
  baseUrl: "http://localhost:8080",
});

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
