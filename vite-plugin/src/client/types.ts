/** Status response from the tygor dev server */
export type TygorStatus =
  | { status: "ok"; port: number; services: string[] }
  | { status: "error"; error: string; phase: "prebuild" | "build" | "runtime"; command: string | null; cwd: string }
  | { status: "reloading" }
  | { status: "starting" }
  | { status: "disconnected" }
  | { status: "vite_disconnected" };

/** RPC error reported by @tygor/client */
export interface TygorRpcError {
  service: string;
  method: string;
  code: string;
  message: string;
  timestamp: number;
}
