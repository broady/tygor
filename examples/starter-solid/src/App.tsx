import { createSignal, onMount, onCleanup } from "solid-js";
import { createClient, ServerError, ValidationError, type StreamState } from "@tygor/client";
import { registry } from "./rpc/manifest";
import { schemaMap } from "./rpc/schemas.map.zod";
import type { MessageState } from "./rpc/types";

const client = createClient(registry, {
  baseUrl: "/api",
  schemas: schemaMap,
  validate: { request: true },
});

function formatError(err: unknown): string {
  if (err instanceof ValidationError) {
    // Client-side validation error - format the zod issues
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

export default function App() {
  const [state, setState] = createSignal<MessageState | null>(null);
  const [connectionState, setConnectionState] = createSignal<StreamState | null>(null);
  const [input, setInput] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);

  // Subscribe to the atom - get current value and updates
  onMount(() => {
    const atom = client.Message.State;
    const unsub = atom.data.subscribe(setState);
    const unsubState = atom.state.subscribe(setConnectionState);
    onCleanup(() => {
      unsub();
      unsubState();
    });
  });

  const handleSet = async (e: Event) => {
    e.preventDefault();
    setError(null);
    try {
      await client.Message.Set({ message: input() });
      setInput("");
    } catch (err) {
      setError(formatError(err));
    }
  };

  return (
    <div>
      <h1>Message Atom</h1>

      {connectionState() && (
        <div style={{ "font-size": "0.75rem", color: connectionState()!.status === "connected" ? "#16a34a" : "#ca8a04", "margin-bottom": "1rem" }}>
          {connectionState()!.status === "connected" ? "\u25CF" : connectionState()!.status === "connecting" ? "\u25CB" : "\u25CC"} {connectionState()!.status}
        </div>
      )}

      {state() && (
        <div style={{ "margin-bottom": "1rem" }}>
          <div style={{ "font-size": "2rem", "font-weight": "bold" }}>{state()!.message}</div>
          <div style={{ "font-size": "0.875rem", color: "#666" }}>
            Set {state()!.set_count} time{state()!.set_count !== 1 ? "s" : ""}
          </div>
        </div>
      )}

      <form onSubmit={handleSet}>
        <input
          value={input()}
          onInput={(e) => setInput(e.currentTarget.value)}
          placeholder="5-10 characters..."
          minLength={5}
          maxLength={10}
        />
        <button type="submit">Set</button>
      </form>

      {error() && (
        <div style={{ color: "#dc2626", "margin-top": "0.5rem", "font-size": "0.875rem" }}>
          {error()}
        </div>
      )}

      <p style={{ "font-size": "0.75rem", color: "#999", "margin-top": "2rem" }}>
        Open this page in multiple tabs - they all sync via the Atom!
      </p>
    </div>
  );
}
