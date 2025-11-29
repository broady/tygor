import { render } from "solid-js/web";
import { createSignal, createEffect, onCleanup, Show } from "solid-js";
import { Overlay } from "./Overlay";
import { StatusIndicator } from "./StatusIndicator";
import { RpcError } from "./RpcError";
import type { TygorStatus, TygorRpcError } from "./types";
import styles from "./styles.css";

const STATUS_DISMISS_DEBOUNCE = 2000; // ms before status can reappear after dismiss
const RPC_ERROR_AUTO_DISMISS = 5000; // ms before RPC error auto-dismisses

function TygorDevtools() {
  const [status, setStatus] = createSignal<TygorStatus | null>(null);
  const [dismissed, setDismissed] = createSignal(false);
  const [lastError, setLastError] = createSignal<string | null>(null);
  const [disconnectedSince, setDisconnectedSince] = createSignal<number | null>(null);
  const [errorSince, setErrorSince] = createSignal<number | null>(null);

  // Status indicator dismiss state
  const [statusDismissedUntil, setStatusDismissedUntil] = createSignal<number>(0);
  const [statusDismissedMessage, setStatusDismissedMessage] = createSignal<string | null>(null);

  // RPC error state
  const [rpcError, setRpcError] = createSignal<TygorRpcError | null>(null);
  let rpcErrorTimeout: ReturnType<typeof setTimeout> | null = null;

  // Reset dismissed state when error changes
  createEffect(() => {
    const s = status();
    if (s?.status === "error") {
      if (s.error !== lastError()) {
        setDismissed(false);
        setLastError(s.error);
        setErrorSince(Date.now());
      }
    } else {
      setErrorSince(null);
    }
  });

  // Listen for RPC errors from @tygor/client
  createEffect(() => {
    const handleRpcError = (event: CustomEvent<TygorRpcError>) => {
      // Clear any pending auto-dismiss
      if (rpcErrorTimeout) clearTimeout(rpcErrorTimeout);

      setRpcError(event.detail);

      // Auto-dismiss after timeout
      rpcErrorTimeout = setTimeout(() => {
        setRpcError(null);
      }, RPC_ERROR_AUTO_DISMISS);
    };

    window.addEventListener("tygor:rpc-error", handleRpcError as EventListener);

    onCleanup(() => {
      window.removeEventListener("tygor:rpc-error", handleRpcError as EventListener);
      if (rpcErrorTimeout) clearTimeout(rpcErrorTimeout);
    });
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
          setStatusDismissedMessage(null); // Reset so next issue shows
        } else if (data.status === "error") {
          setDisconnectedSince(null);
        } else {
          // reloading, starting, disconnected
          if (disconnectedSince() === null) {
            setDisconnectedSince(Date.now());
          }
        }
      } catch {
        // Fetch failed = Vite dev server is down (not Go server)
        if (disconnectedSince() === null) {
          setDisconnectedSince(Date.now());
        }
        setStatus({ status: "vite_disconnected" });
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
      case "vite_disconnected":
        return "Vite dev server disconnected";
      default:
        return "";
    }
  };

  const shouldShowStatus = () => {
    const s = status();
    if (!s) return false;
    if (s.status === "error") return false; // Overlay handles errors
    if (s.status === "ok") return false; // All good, nothing to show

    const msg = statusMessage();
    const now = Date.now();

    // If same message as dismissed, stay dismissed until message changes
    if (msg === statusDismissedMessage()) {
      return false;
    }

    // Different message - only show if debounce period has passed
    // (prevents flicker during rapid status changes)
    if (statusDismissedMessage() !== null && now < statusDismissedUntil()) {
      return false;
    }

    return true;
  };

  const handleStatusDismiss = () => {
    setStatusDismissedUntil(Date.now() + STATUS_DISMISS_DEBOUNCE);
    setStatusDismissedMessage(statusMessage());
  };

  const handleRpcErrorDismiss = () => {
    if (rpcErrorTimeout) clearTimeout(rpcErrorTimeout);
    setRpcError(null);
  };

  return (
    <>
      <Show when={shouldShowOverlay()}>
        <Overlay
          status={status() as TygorStatus & { status: "error" }}
          onDismiss={() => setDismissed(true)}
          timestamp={errorSince() ?? undefined}
        />
      </Show>
      <Show when={!shouldShowOverlay() && rpcError()}>
        <RpcError error={rpcError()!} onDismiss={handleRpcErrorDismiss} />
      </Show>
      <Show when={!shouldShowOverlay() && !rpcError() && shouldShowStatus()}>
        <StatusIndicator
          message={statusMessage()}
          timestamp={disconnectedSince()}
          onDismiss={handleStatusDismiss}
        />
      </Show>
    </>
  );
}

// Mount the devtools in Shadow DOM for style isolation
const host = document.createElement("div");
host.id = "tygor-devtools";
document.body.appendChild(host);

const shadow = host.attachShadow({ mode: "open" });

// Inject styles into shadow root
const styleEl = document.createElement("style");
styleEl.textContent = styles;
shadow.appendChild(styleEl);

// Render into shadow root
const container = document.createElement("div");
shadow.appendChild(container);
render(() => <TygorDevtools />, container);
