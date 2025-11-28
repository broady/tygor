# React Example

A minimal React + Vite application demonstrating tygor's type-safe RPC client.

## Features

- React 19 with Vite
- Type-safe API calls with generated TypeScript
- Simple todo list with create/toggle operations

## Quick Start

```bash
# Terminal 1: Start Go server
make run

# Terminal 2: Start Vite dev server
make dev
```

Open http://localhost:5173 to see the app.

## Project Structure

```
react/
├── main.go           # Go server with handlers
├── api/types.go      # Request/response types
└── client/
    ├── src/
    │   ├── App.tsx   # React component
    │   └── rpc/      # Generated types
    └── vite.config.js
```

## Code Snippets

### Go Handlers

<!-- [snippet:handlers] -->
```go title="main.go"
func ListTasks(ctx context.Context, req *api.ListTasksParams) ([]*api.Task, error) {
	tasksMu.Lock()
	defer tasksMu.Unlock()

	if req.ShowDone != nil && !*req.ShowDone {
		filtered := []*api.Task{}
		for _, t := range tasks {
			if !t.Done {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	}
	if tasks == nil {
		return []*api.Task{}, nil
	}
	return tasks, nil
}

func CreateTask(ctx context.Context, req *api.CreateTaskParams) (*api.Task, error) {
	tasksMu.Lock()
	defer tasksMu.Unlock()

	task := &api.Task{
		ID:    nextID,
		Title: req.Title,
		Done:  false,
	}
	nextID++
	tasks = append(tasks, task)
	return task, nil
}

func ToggleTask(ctx context.Context, req *api.ToggleTaskParams) (*api.Task, error) {
	tasksMu.Lock()
	defer tasksMu.Unlock()

	for _, t := range tasks {
		if t.ID == req.ID {
			t.Done = !t.Done
			return t, nil
		}
	}
	return nil, tygor.NewError(tygor.CodeNotFound, "task not found")
}

```
<!-- [/snippet:handlers] -->

### App Setup

<!-- [snippet:app-setup] -->
```go title="main.go"
app := tygor.NewApp().
	WithMiddleware(middleware.CORS(&middleware.CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

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
import type { Task } from "./rpc/types";

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTasks = async () => {
    try {
      const result = await client.Tasks.List({});
      setTasks(result);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to fetch tasks");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTasks();
  }, []);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!newTask.trim()) return;

    try {
      await client.Tasks.Create({ title: newTask });
      setNewTask("");
      setError(null);
      fetchTasks();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to create task");
    }
  };

  const handleToggle = async (id: number) => {
    try {
      await client.Tasks.Toggle({ id });
      setError(null);
      fetchTasks();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to toggle task");
    }
  };

  return (
    <div style={{ maxWidth: 400, margin: "2rem auto", fontFamily: "system-ui" }}>
      <h1>Tasks</h1>

      {error && (
        <div style={{ padding: "0.5rem", marginBottom: "1rem", background: "#fee", color: "#c00", borderRadius: 4 }}>
          {error}
        </div>
      )}

      {loading ? (
        <div>Loading...</div>
      ) : (
        <>
          <form onSubmit={handleCreate} style={{ marginBottom: "1rem" }}>
            <input
              value={newTask}
              onChange={(e) => setNewTask(e.target.value)}
              placeholder="New task..."
              style={{ padding: "0.5rem", marginRight: "0.5rem" }}
            />
            <button type="submit" style={{ padding: "0.5rem 1rem" }}>
              Add
            </button>
          </form>

          <ul style={{ listStyle: "none", padding: 0 }}>
            {tasks.map((task) => (
              <li
                key={task.id}
                onClick={() => handleToggle(task.id)}
                style={{
                  padding: "0.5rem",
                  cursor: "pointer",
                  textDecoration: task.done ? "line-through" : "none",
                  opacity: task.done ? 0.6 : 1,
                }}
              >
                {task.done ? "✓" : "○"} {task.title}
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}

```
<!-- [/snippet:react-component] -->

## Generated Types

<!-- [snippet-file:client/src/rpc/types.ts] -->
```typescript title="types.ts"
// Code generated by tygor. DO NOT EDIT.

export * from './types_github_com_broady_tygor_examples_react_api';
```
