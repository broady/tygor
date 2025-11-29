import { Show, createSignal, onCleanup } from "solid-js";

function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  return `${Math.floor(minutes / 60)}h`;
}

interface StatusIndicatorProps {
  message: string;
  disconnectedSince: number | null;
}

export function StatusIndicator(props: StatusIndicatorProps) {
  // Tick signal to force duration recalculation every second
  const [tick, setTick] = createSignal(0);
  const interval = setInterval(() => setTick((t) => t + 1), 1000);
  onCleanup(() => clearInterval(interval));

  const duration = () => {
    tick(); // Subscribe to tick updates
    if (!props.disconnectedSince) return null;
    return formatDuration(Date.now() - props.disconnectedSince);
  };

  return (
    <div class="tygor-status">
      <span class="tygor-status-prefix">[tygor]</span> {props.message}
      <Show when={duration()}>
        <span class="tygor-status-duration"> ({duration()})</span>
      </Show>
    </div>
  );
}
