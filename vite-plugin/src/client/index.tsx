import { render } from "solid-js/web";
import { createSignal, createEffect, onCleanup, Show } from "solid-js";
import { Overlay } from "./Overlay";
import { StatusIndicator } from "./StatusIndicator";
import type { TygorStatus } from "./types";
import styles from "./styles.css";

// Inject styles
const styleEl = document.createElement("style");
styleEl.textContent = styles;
document.head.appendChild(styleEl);

function TygorDevtools() {
  const [status, setStatus] = createSignal<TygorStatus | null>(null);
  const [dismissed, setDismissed] = createSignal(false);
  const [lastError, setLastError] = createSignal<string | null>(null);
  const [disconnectedSince, setDisconnectedSince] = createSignal<number | null>(null);

  // Reset dismissed state when error changes
  createEffect(() => {
    const s = status();
    if (s?.status === "error") {
      if (s.error !== lastError()) {
        setDismissed(false);
        setLastError(s.error);
      }
    }
  });

  // Polling effect
  createEffect(() => {
    let mounted = true;

    async function poll() {
      if (!mounted) return;

      try {
        const res = await fetch("/__tygor/status");
        const data: TygorStatus = await res.json();
        setStatus(data);

        if (data.status === "ok") {
          setDisconnectedSince(null);
          setLastError(null);
          setDismissed(false);
        } else if (data.status === "error") {
          setDisconnectedSince(null);
        } else {
          // reloading, starting, disconnected
          if (disconnectedSince() === null) {
            setDisconnectedSince(Date.now());
          }
        }
      } catch {
        if (disconnectedSince() === null) {
          setDisconnectedSince(Date.now());
        }
        setStatus({ status: "disconnected" });
      }

      if (mounted) {
        setTimeout(poll, 1000);
      }
    }

    poll();

    onCleanup(() => {
      mounted = false;
    });
  });

  const shouldShowOverlay = () => {
    const s = status();
    return s?.status === "error" && !dismissed();
  };

  const shouldShowStatus = () => {
    const s = status();
    if (!s) return false;
    if (s.status === "error") return false; // Overlay handles errors
    if (s.status === "ok") return false; // All good, nothing to show
    return true;
  };

  const statusMessage = () => {
    const s = status();
    if (!s) return "";
    switch (s.status) {
      case "reloading":
        return "Reloading Go server...";
      case "starting":
        return "Starting Go server...";
      case "disconnected":
        return "Go server disconnected";
      default:
        return "";
    }
  };

  return (
    <>
      <Show when={shouldShowOverlay()}>
        <Overlay
          status={status() as TygorStatus & { status: "error" }}
          onDismiss={() => setDismissed(true)}
        />
      </Show>
      <Show when={shouldShowStatus()}>
        <StatusIndicator message={statusMessage()} disconnectedSince={disconnectedSince()} />
      </Show>
    </>
  );
}

// Mount the devtools
const container = document.createElement("div");
container.id = "tygor-devtools";
document.body.appendChild(container);
render(() => <TygorDevtools />, container);
