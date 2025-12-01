// [snippet:client-setup]

import { useState, useEffect, FormEvent } from "react";
import { createClient } from "@tygor/client";
import { registry } from "./rpc/manifest";
import type { Task, RuntimeInfo } from "./rpc/types";

// No baseUrl needed - uses current origin (works with Vite proxy in dev, same-origin in prod)
const client = createClient(registry);

// [/snippet:client-setup]

// [snippet:react-component]

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

  useEffect(() => client.System.InfoStream().subscribe(setInfo), []);

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

// [/snippet:react-component]
