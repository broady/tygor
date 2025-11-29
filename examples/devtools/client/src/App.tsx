import { useState, useEffect, FormEvent } from "react";
import { createClient } from "@tygor/client";
import { registry } from "./rpc/manifest";
import type { Task, InfoResponse } from "./rpc/types";

// No baseUrl needed - uses current origin (works with Vite proxy in dev, same-origin in prod)
const client = createClient(registry);

export default function App() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [newTask, setNewTask] = useState("");
  const [info, setInfo] = useState<InfoResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetchTasks = async () => {
    try {
      setTasks(await client.Tasks.List({}));
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Server unavailable");
    }
  };

  useEffect(() => {
    fetchTasks();
  }, []);

  useEffect(() => {
    const fetchInfo = async () => {
      try {
        setInfo(await client.Devtools.Info({}));
        setError(null);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Server unavailable");
      }
    };
    fetchInfo();
    const interval = setInterval(fetchInfo, 5000);
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
      {error ? (
        <div className="info" style={{ background: "#fff3cd", color: "#856404" }}>
          Connecting to server...
        </div>
      ) : info && (
        <div className="info">
          <strong>:{info.port}</strong> | {info.version} | {info.num_goroutines} goroutines |{" "}
          {formatBytes(info.memory.alloc)} alloc | {info.memory.num_gc} GC |{" "}
          <button onClick={() => client.System.Kill({})} style={{ fontSize: "0.7rem", padding: "2px 6px" }}>
            Kill
          </button>
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
