import { useState, useEffect, FormEvent, Component, ReactNode } from "react";
import { createClient, ServerError, TransportError } from "@tygor/client";
import type { SubscriptionResult } from "@tygor/client";
import { registry } from "./rpc/manifest";
import type { Task, TimeUpdate } from "./rpc/types";

const client = createClient(registry, {
  baseUrl: "/api",
});

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
  const [tasks, setTasks] = useState<(Task | null)[]>([]);
  const [newTask, setNewTask] = useState("");
  const [error, setError] = useState<AppError | null>(null);
  const [time, setTime] = useState<TimeUpdate | null>(null);
  const [tasksState, setTasksState] = useState<SubscriptionResult<(Task | null)[]> | null>(null);

  // Subscribe to synced task list - automatically updates across all browser windows
  useEffect(() => {
    return client.Tasks.SyncedList.subscribe((result) => {
      setTasks(result.data ?? []);
      setTasksState(result);
    });
  }, []);

  // Subscribe to time stream
  useEffect(() => {
    const stream = client.Tasks.Time({});
    return stream.subscribe((result) => {
      if (result.data) setTime(result.data);
    });
  }, []);


  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!newTask.trim()) return;
    try {
      await client.Tasks.Create({ title: newTask });
      setNewTask("");
      setError(null);
    } catch (e) {
      setError(toAppError(e));
    }
  };

  const handleToggle = async (id: number) => {
    try {
      await client.Tasks.Toggle({ id });
      setError(null);
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
        {tasksState && (
          <div style={{ fontSize: "0.7rem", color: tasksState.status === "connected" ? "#16a34a" : "#ca8a04", marginBottom: 8 }}>
            {tasksState.status === "connected" ? "●" : tasksState.status === "connecting" ? "○" : "◌"} {tasksState.status}
          </div>
        )}
        {time && (
          <div style={{ fontSize: "0.8rem", color: "#666", marginBottom: 12, fontVariantNumeric: "tabular-nums" }}>
            Server time: {new Date(time.time).toLocaleTimeString()}{" "}
            <span style={{ opacity: 0.6 }}>(via Tasks.Time SSE)</span>
          </div>
        )}
        <form onSubmit={handleCreate}>
          <input
            value={newTask}
            onChange={(e) => setNewTask(e.target.value)}
            placeholder="New task..."
          />
          <button type="submit">Add</button>
        </form>
        <ul>
          {tasks.filter((t): t is Task => t !== null).map((task) => (
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
