import { createSignal, onMount, onCleanup } from "solid-js";
import type { Atom, SubscriptionResult } from "@tygor/client";

/**
 * Solid.js hook for subscribing to a tygor Atom or Stream.
 * Returns an accessor that provides the current SubscriptionResult.
 *
 * @example
 * const result = useAtom(client.Message.State);
 * // In JSX:
 * <Show when={result().data}>
 *   {(data) => <div>{data().message}</div>}
 * </Show>
 */
export function useAtom<T>(atom: Atom<T>) {
  const [state, setState] = createSignal<SubscriptionResult<T>>(atom.getSnapshot());

  onMount(() => {
    const unsub = atom.subscribe(setState);
    onCleanup(unsub);
  });

  return state;
}
