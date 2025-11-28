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

  const fetchTasks = async () => {
    setTasks(await client.Tasks.List({}));
  };

  useEffect(() => {
    fetchTasks();
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

  return (
    <div>
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
