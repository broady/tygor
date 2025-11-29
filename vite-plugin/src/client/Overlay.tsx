import { createSignal, Show } from "solid-js";
import type { TygorStatus } from "./types";

const PHASE_LABELS: Record<string, string> = {
  prebuild: "Prebuild",
  build: "Build",
  runtime: "Runtime",
};

interface OverlayProps {
  status: TygorStatus & { status: "error" };
  onDismiss: () => void;
}

export function Overlay(props: OverlayProps) {
  const [copied, setCopied] = createSignal(false);

  const title = () => `tygor: Go ${PHASE_LABELS[props.status.phase] || "Build"} Error`;

  const copyText = () => {
    const { command, cwd, error } = props.status;
    return (command ? `${cwd}$ ${command}\n\n` : "") + error.trim();
  };

  const handleCopy = async () => {
    await navigator.clipboard.writeText(copyText());
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div class="tygor-overlay">
      <div class="tygor-overlay-header">
        <h2 class="tygor-overlay-title">{title()}</h2>
        <div class="tygor-overlay-actions">
          <button
            class={`tygor-btn ${copied() ? "tygor-btn-copied" : ""}`}
            onClick={handleCopy}
            title="Copy error"
          >
            {copied() ? "Copied" : "Copy"}
          </button>
          <button class="tygor-btn" onClick={props.onDismiss} title="Dismiss">
            ×
          </button>
        </div>
      </div>
      <div class="tygor-overlay-body">
        <Show when={props.status.command}>
          <p class="tygor-overlay-command">$ {props.status.command}</p>
        </Show>
        <pre class="tygor-overlay-error">{props.status.error}</pre>
        <p class="tygor-overlay-hint">Fix the error and save — auto-reloads when fixed.</p>
      </div>
    </div>
  );
}
