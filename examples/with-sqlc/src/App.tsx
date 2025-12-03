// [snippet:client-setup]

import { createSignal, createEffect, on, For, Show, Switch, Match, createResource } from "solid-js";
import { createClient, ServerError, ValidationError } from "@tygor/client";
import { registry } from "./rpc/manifest";
import { schemaMap } from "./rpc/schemas.map.zod";
import type { Task } from "./rpc/types";
import { useAtom } from "./useAtom";
import styles from "./App.module.css";

const client = createClient(registry, {
  baseUrl: "/api",
  schemas: schemaMap,
  validate: { request: true },
});
// [/snippet:client-setup]

function formatError(err: unknown): string {
  if (err instanceof ValidationError) {
    const messages = err.issues.map((issue) => {
      const path = issue.path?.join(".") ?? "";
      return path ? `${path}: ${issue.message}` : issue.message;
    });
    return messages.join("; ");
  }
  if (err instanceof ServerError) {
    return err.message;
  }
  return err instanceof Error ? err.message : "Unknown error";
}

const TYPE_CHAIN = ["SQL", "sqlc", "Go", "tygor", "TypeScript"];
const UI_STACK = ["Solid", "Vite"];

// [snippet:client-usage]

export default function App() {
  const [tasks, { refetch }] = createResource(() =>
    client.Tasks.List({ limit: 50, offset: 0 })
  );
  const [title, setTitle] = createSignal("");
  const [description, setDescription] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);
  const [streaming, setStreaming] = createSignal(false);

  // Subscribe to version changes - refetch when version bumps
  const version = useAtom(client.Tasks.Version);
  createEffect(
    on(
      () => version().data?.value,
      () => { if (streaming()) refetch(); },
      { defer: true }
    )
  );

  const handleCreate = async (e: Event) => {
    e.preventDefault();
    setError(null);
    try {
      await client.Tasks.Create({ title: title(), description: description() });
      setTitle("");
      setDescription("");
      if (!streaming()) refetch();
    } catch (err) {
      setError(formatError(err));
    }
  };

  const handleToggle = async (task: Task) => {
    try {
      await client.Tasks.Update({
        id: task.id,
        title: task.title,
        description: task.description,
        completed: task.completed ? 0 : 1, // SQLite uses int for bool
      });
      if (!streaming()) refetch();
    } catch (err) {
      setError(formatError(err));
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await client.Tasks.Delete({ id });
      if (!streaming()) refetch();
    } catch (err) {
      setError(formatError(err));
    }
  };
// [/snippet:client-usage]

  return (
    <div class={styles.app}>
      <h1 class={styles.title}>Tygor Tasks</h1>
      <div class={styles.tagline}>
        <span class={styles.taglineLabel}>Type-safe:</span>
        <For each={TYPE_CHAIN}>
          {(step, i) => (
            <>
              <span class={styles.taglineStep}>{step}</span>
              {i() < TYPE_CHAIN.length - 1 && (
                <span class={styles.taglineArrow}>&rarr;</span>
              )}
            </>
          )}
        </For>
        <span class={styles.taglineSeparator}>|</span>
        <span class={styles.taglineLabel}>UI:</span>
        <For each={UI_STACK}>
          {(step, i) => (
            <>
              <span class={styles.taglineStep}>{step}</span>
              {i() < UI_STACK.length - 1 && (
                <span class={styles.taglineArrow}>+</span>
              )}
            </>
          )}
        </For>
      </div>

      <div class={styles.controls}>
        <label class={styles.toggle}>
          <input
            type="checkbox"
            checked={streaming()}
            onChange={(e) => setStreaming(e.currentTarget.checked)}
          />
          Live updates
        </label>
        <Show when={streaming()}>
          <span
            class={styles.status}
            classList={{
              [styles.statusConnected]: version().isConnected,
              [styles.statusConnecting]: !version().isConnected,
            }}
          >
            <Switch fallback="◌ disconnected">
              <Match when={version().isConnected}>● connected</Match>
              <Match when={version().isConnecting}>○ connecting...</Match>
            </Switch>
          </span>
        </Show>
      </div>

      <form onSubmit={handleCreate} class={styles.form}>
        <input
          class={styles.input}
          value={title()}
          onInput={(e) => setTitle(e.currentTarget.value)}
          placeholder="Task title"
        />
        <input
          class={styles.input}
          value={description()}
          onInput={(e) => setDescription(e.currentTarget.value)}
          placeholder="Description (optional)"
        />
        <button type="submit" class={styles.button}>
          Add Task
        </button>
      </form>

      <Show when={error()}>
        <div class={styles.error}>{error()}</div>
      </Show>

      <Show when={tasks.loading && !tasks.latest}>
        <p>Loading...</p>
      </Show>

      <Show when={tasks.latest}>
        <ul class={styles.list}>
          <For each={tasks.latest}>
            {(task) => (
              <li class={styles.task}>
                <input
                  type="checkbox"
                  checked={!!task.completed}
                  onChange={() => handleToggle(task)}
                />
                <div class={styles.taskContent}>
                  <div
                    class={styles.taskTitle}
                    classList={{ [styles.taskTitleCompleted]: !!task.completed }}
                  >
                    {task.title}
                  </div>
                  <Show when={task.description}>
                    <div class={styles.taskDescription}>
                      {task.description}
                    </div>
                  </Show>
                </div>
                <button
                  class={styles.taskDelete}
                  onClick={() => handleDelete(task.id)}
                >
                  ✕
                </button>
              </li>
            )}
          </For>
        </ul>
        <Show when={tasks.latest?.length === 0}>
          <p class={styles.empty}>No tasks yet</p>
        </Show>
      </Show>
    </div>
  );
}
