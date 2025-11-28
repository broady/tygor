// [snippet:client-setup]

import { useState, useEffect, FormEvent } from "react";
import { createClient } from "@tygor/client";
import { registry } from "./rpc/manifest";
import type { Task } from "./rpc/types";

const client = createClient(registry, {
  baseUrl: "http://localhost:8080",
});

// [/snippet:client-setup]

// [snippet:react-component]

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

// [/snippet:react-component]
