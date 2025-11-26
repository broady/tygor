/**
 * Shared test utilities for tygor example integration tests.
 * Provides server lifecycle management for Bun tests.
 */

import type { Subprocess } from "bun";

export interface ServerOptions {
  /** Directory containing main.go */
  cwd: string;
  /** Port to use (default: random available) */
  port?: number;
  /** Milliseconds to wait for server startup (default: 5000) */
  startupTimeout?: number;
}

export interface RunningServer {
  url: string;
  port: number;
  process: Subprocess;
  stop: () => Promise<void>;
}

/**
 * Find an available port by binding to port 0.
 */
async function findAvailablePort(): Promise<number> {
  const server = Bun.listen({
    hostname: "127.0.0.1",
    port: 0,
    socket: {
      data() {},
    },
  });
  const port = server.port;
  server.stop();
  return port;
}

/**
 * Wait for a server to be ready by polling.
 */
async function waitForServer(url: string, timeout: number): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      await fetch(url);
      return true;
    } catch {
      // Connection refused, server not ready yet
      await Bun.sleep(50);
    }
  }
  return false;
}

/**
 * Start a Go server and wait for it to be ready.
 *
 * @example
 * ```ts
 * let server: RunningServer;
 *
 * beforeAll(async () => {
 *   server = await startServer({
 *     cwd: new URL("../../", import.meta.url).pathname,
 *   });
 * });
 *
 * afterAll(async () => {
 *   await server?.stop();
 * });
 * ```
 */
export async function startServer(options: ServerOptions): Promise<RunningServer> {
  const port = options.port ?? (await findAvailablePort());
  const startupTimeout = options.startupTimeout ?? 5000;

  // Build first for faster startup and better error messages
  const build = Bun.spawnSync(["go", "build", "-o", "server", "."], {
    cwd: options.cwd,
    stdout: "inherit",
    stderr: "inherit",
  });

  if (build.exitCode !== 0) {
    throw new Error(`Failed to build server in ${options.cwd}`);
  }

  // Start the server
  const process = Bun.spawn(["./server", "-port", String(port)], {
    cwd: options.cwd,
    stdout: "pipe",
    stderr: "pipe",
  });

  const url = `http://localhost:${port}`;

  // Wait for server to be ready
  const ready = await waitForServer(url, startupTimeout);
  if (!ready) {
    process.kill();
    throw new Error(`Server failed to start within ${startupTimeout}ms`);
  }

  return {
    url,
    port,
    process,
    stop: async () => {
      process.kill();
      await process.exited;
    },
  };
}
