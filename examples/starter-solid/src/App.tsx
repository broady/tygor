import { createSignal, Show } from "solid-js";
import { createClient, ServerError, ValidationError } from "@tygor/client";
import { registry } from "./rpc/manifest";
import { schemaMap } from "./rpc/schemas.map.zod";
import { useAtom } from "./useAtom";

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
  const atom = useAtom(client.Message.State);
  const [input, setInput] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);

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

      <div style={{ "font-size": "0.75rem", color: atom().isConnected ? "#16a34a" : "#ca8a04", "margin-bottom": "1rem" }}>
        {atom().isConnected ? "\u25CF" : atom().isConnecting ? "\u25CB" : "\u25CC"} {atom().status}
      </div>

      <Show when={atom().data}>
        {(data) => (
          <div style={{ "margin-bottom": "1rem" }}>
            <div style={{ "font-size": "2rem", "font-weight": "bold" }}>{data().message}</div>
            <div style={{ "font-size": "0.875rem", color: "#666" }}>
              Set {data().set_count} time{data().set_count !== 1 ? "s" : ""}
            </div>
          </div>
        )}
      </Show>

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

      <Show when={error()}>
        <div style={{ color: "#dc2626", "margin-top": "0.5rem", "font-size": "0.875rem" }}>
          {error()}
        </div>
      </Show>

      <p style={{ "font-size": "0.75rem", color: "#999", "margin-top": "2rem" }}>
        Open this page in multiple tabs - they all sync via the Atom!
      </p>
    </div>
  );
}
