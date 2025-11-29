import { createSignal, createEffect, onCleanup, Show } from "solid-js";
import { TigerButton, extractErrorSummary } from "./TigerButton";
import { Sidebar } from "./Sidebar";
import type { TygorStatus, TygorRpcError } from "./types";

export type DevToolsMode = "overlay" | "sidebar";
export type SidebarSide = "left" | "right";

interface DevToolsState {
  status: TygorStatus | null;
  rpcError: TygorRpcError | null;
  disconnectedSince: number | null;
  errorSince: number | null;
}

const RPC_ERROR_AUTO_DISMISS = 5000;
const SIDEBAR_WIDTH = 300;
const DOCKED_STYLE_ID = "tygor-docked-styles";

export function DevTools() {
  const [mode, setMode] = createSignal<DevToolsMode>("overlay");
  const [docked, setDocked] = createSignal(false);
  const [side] = createSignal<SidebarSide>("right");
  const [state, setState] = createSignal<DevToolsState>({
    status: null,
    rpcError: null,
    disconnectedSince: null,
    errorSince: null,
  });

  let rpcErrorTimeout: ReturnType<typeof setTimeout> | null = null;

  // Manage docked styles in the main document (outside shadow DOM)
  createEffect(() => {
    const isDocked = docked() && mode() === "sidebar";
    const currentSide = side();

    let styleEl = document.getElementById(DOCKED_STYLE_ID) as HTMLStyleElement | null;

    if (isDocked) {
      if (!styleEl) {
        styleEl = document.createElement("style");
        styleEl.id = DOCKED_STYLE_ID;
        document.head.appendChild(styleEl);
      }

      const prop = currentSide === "right" ? "margin-right" : "margin-left";

      styleEl.textContent = `
        /* Tygor DevTools: Docked mode - shifts page content */
        html {
          ${prop}: ${SIDEBAR_WIDTH}px !important;
          transition: ${prop} 0.2s ease-out;
        }
      `;
    } else {
      styleEl?.remove();
    }
  });

  // Cleanup docked styles on unmount
  onCleanup(() => {
    document.getElementById(DOCKED_STYLE_ID)?.remove();
  });

  // Poll status from server
  createEffect(() => {
    let mounted = true;

    async function poll() {
      if (!mounted) return;

      try {
        const res = await fetch("/__tygor/status");
        const data: TygorStatus = await res.json();

        setState((prev) => {
          const next = { ...prev, status: data };

          if (data.status === "ok") {
            next.disconnectedSince = null;
          } else if (data.status === "error") {
            next.disconnectedSince = null;
            if (prev.status?.status !== "error" || (prev.status?.status === "error" && prev.status.error !== data.error)) {
              next.errorSince = Date.now();
            }
          } else {
            // reloading, starting, disconnected
            if (prev.disconnectedSince === null) {
              next.disconnectedSince = Date.now();
            }
          }

          return next;
        });
      } catch {
        setState((prev) => ({
          ...prev,
          status: { status: "vite_disconnected" },
          disconnectedSince: prev.disconnectedSince ?? Date.now(),
        }));
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

  // Listen for RPC errors
  createEffect(() => {
    const handleRpcError = (event: CustomEvent<TygorRpcError>) => {
      if (rpcErrorTimeout) clearTimeout(rpcErrorTimeout);

      setState((prev) => ({ ...prev, rpcError: event.detail }));

      rpcErrorTimeout = setTimeout(() => {
        setState((prev) => ({ ...prev, rpcError: null }));
      }, RPC_ERROR_AUTO_DISMISS);
    };

    window.addEventListener("tygor:rpc-error", handleRpcError as EventListener);

    onCleanup(() => {
      window.removeEventListener("tygor:rpc-error", handleRpcError as EventListener);
      if (rpcErrorTimeout) clearTimeout(rpcErrorTimeout);
    });
  });

  const isBuilding = () => {
    const s = state().status;
    return s?.status === "reloading" || s?.status === "starting";
  };

  const hasError = () => {
    const s = state().status;
    return s?.status === "error";
  };

  const isDisconnected = () => {
    const s = state().status;
    return s?.status === "disconnected" || s?.status === "vite_disconnected";
  };

  const errorInfo = () => {
    const s = state().status;
    if (s?.status !== "error") return null;
    return {
      phase: s.phase,
      summary: extractErrorSummary(s.error, s.phase),
      exitCode: s.exitCode,
    };
  };

  const toggleMode = () => {
    setMode((m) => (m === "overlay" ? "sidebar" : "overlay"));
  };

  const toggleDocked = () => {
    setDocked((d) => !d);
  };

  const dismissRpcError = () => {
    if (rpcErrorTimeout) clearTimeout(rpcErrorTimeout);
    setState((prev) => ({ ...prev, rpcError: null }));
  };

  return (
    <>
      <Show when={mode() === "overlay"}>
        <TigerButton
          isBuilding={isBuilding()}
          hasError={hasError()}
          isDisconnected={isDisconnected()}
          errorInfo={errorInfo()}
          onClick={toggleMode}
        />
      </Show>
      <Show when={mode() === "sidebar"}>
        <Sidebar
          state={state()}
          side={side()}
          docked={docked()}
          onCollapse={toggleMode}
          onToggleDocked={toggleDocked}
          onDismissRpcError={dismissRpcError}
        />
      </Show>
    </>
  );
}
