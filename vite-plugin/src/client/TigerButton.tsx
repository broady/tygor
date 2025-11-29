import { Show } from "solid-js";

interface ErrorInfo {
  phase: string;
  summary: string | null;
}

interface TigerButtonProps {
  isBuilding: boolean;
  hasError: boolean;
  isDisconnected: boolean;
  errorInfo: ErrorInfo | null;
  onClick: () => void;
}

const PHASE_LABELS: Record<string, string> = {
  prebuild: "Prebuild Error",
  build: "Build Error",
  runtime: "Runtime Error",
};

/** Extract a one-line summary from Go build output */
function extractErrorSummary(error: string): string | null {
  // Match Go compiler error: ./file.go:line:col: message
  const goErrorMatch = error.match(/\.\/([^:]+):(\d+):\d+:\s*(.+)/);
  if (goErrorMatch) {
    const [, file, line, message] = goErrorMatch;
    // Truncate message if too long
    const shortMsg = message.length > 40 ? message.slice(0, 37) + "..." : message;
    return `${file}:${line} ${shortMsg}`;
  }

  // Match general first line of error
  const firstLine = error.trim().split("\n")[0];
  if (firstLine && firstLine.length < 60) {
    return firstLine;
  }

  return null;
}

export function TigerButton(props: TigerButtonProps) {
  const showRunning = () => props.isBuilding && !props.isDisconnected;
  const showExpanded = () => props.hasError && !props.isDisconnected && props.errorInfo;

  const phaseLabel = () => {
    if (!props.errorInfo) return "Error";
    return PHASE_LABELS[props.errorInfo.phase] || "Error";
  };

  return (
    <button
      class="tygor-tiger-btn"
      classList={{
        "tygor-tiger-btn--building": showRunning(),
        "tygor-tiger-btn--error": props.hasError && !props.isDisconnected,
        "tygor-tiger-btn--disconnected": props.isDisconnected,
        "tygor-tiger-btn--expanded": showExpanded(),
      }}
      onClick={props.onClick}
      title={props.isDisconnected ? "Tygor DevTools (disconnected)" : "Open Tygor DevTools"}
    >
      <span class="tygor-tiger-btn-icon">
        <Show when={showRunning()} fallback={<span class="tygor-tiger-icon">🐯</span>}>
          <span class="tygor-tiger-runner">🐅</span>
        </Show>
      </span>
      <Show when={showExpanded()}>
        <span class="tygor-tiger-btn-content">
          <span class="tygor-tiger-btn-title">{phaseLabel()}</span>
          <Show when={props.errorInfo?.summary}>
            <span class="tygor-tiger-btn-summary">{props.errorInfo!.summary}</span>
          </Show>
        </span>
      </Show>
    </button>
  );
}

export { extractErrorSummary };
