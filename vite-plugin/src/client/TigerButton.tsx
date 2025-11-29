import { Show } from "solid-js";

interface TigerButtonProps {
  isBuilding: boolean;
  hasError: boolean;
  isDisconnected: boolean;
  onClick: () => void;
}

export function TigerButton(props: TigerButtonProps) {
  const showRunning = () => props.isBuilding && !props.isDisconnected;

  return (
    <button
      class="tygor-tiger-btn"
      classList={{
        "tygor-tiger-btn--building": showRunning(),
        "tygor-tiger-btn--error": props.hasError && !props.isDisconnected,
        "tygor-tiger-btn--disconnected": props.isDisconnected,
      }}
      onClick={props.onClick}
      title={props.isDisconnected ? "Tygor DevTools (disconnected)" : "Open Tygor DevTools"}
    >
      <Show when={showRunning()} fallback={<span class="tygor-tiger-icon">🐯</span>}>
        <span class="tygor-tiger-runner">🐅</span>
      </Show>
    </button>
  );
}
