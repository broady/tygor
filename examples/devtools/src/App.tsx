import { useState, useEffect, FormEvent, Component, ReactNode } from "react";
import { createClient, ServerError, TransportError } from "@tygor/client";
import { registry } from "./rpc/manifest";
import type { Task } from "./rpc/types";

// No baseUrl needed - uses current origin (works with Vite proxy in dev, same-origin in prod)
const client = createClient(registry);

interface AppError {
  type: "server" | "transport" | "unknown";
  code?: string;
  message: string;
}

function toAppError(e: unknown): AppError {
  if (e instanceof ServerError) {
    return { type: "server", code: e.code, message: e.message };
  }
  if (e instanceof TransportError) {
    return { type: "transport", message: e.message };
  }
  return { type: "unknown", message: e instanceof Error ? e.message : "Unknown error" };
}

// ErrorBoundary for catching render errors
interface ErrorBoundaryState {
  error: Error | null;
}

class ErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 20, background: "#fee2e2", color: "#991b1b", borderRadius: 4, margin: 16 }}>
          <strong>React Error</strong>
          <pre style={{ margin: "8px 0 0", fontSize: 12, whiteSpace: "pre-wrap" }}>
            {this.state.error.message}
          </pre>
          <button
            onClick={() => this.setState({ error: null })}
            style={{ marginTop: 8, fontSize: 12, padding: "4px 8px" }}
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

export default function App() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [newTask, setNewTask] = useState("");
  const [error, setError] = useState<AppError | null>(null);

  const fetchTasks = async () => {
    try {
      setTasks(await client.Tasks.List());
      setError(null);
    } catch (e) {
      setError(toAppError(e));
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
      setError(toAppError(e));
    }
  };

  const handleToggle = async (id: number) => {
    try {
      await client.Tasks.Toggle({ id });
      setError(null);
      fetchTasks();
    } catch (e) {
      setError(toAppError(e));
    }
  };

  return (
    <ErrorBoundary>
      <div>
        {error && (
          <div className="info" style={{ background: "#fff3cd", color: "#856404" }}>
            <strong>{error.type}</strong>
            {error.code && <code style={{ marginLeft: 8, marginRight: 8, background: "rgba(0,0,0,0.1)", padding: "2px 6px", borderRadius: 3 }}>{error.code}</code>}
            <span style={{ marginLeft: 8 }}>{error.message}</span>
          </div>
        )}
        <button onClick={() => client.Tasks.MakeError().catch((e) => setError(toAppError(e)))} style={{ fontSize: "0.7rem", padding: "2px 6px" }}>
          Make an error
        </button>

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
    </ErrorBoundary>
  );
}
