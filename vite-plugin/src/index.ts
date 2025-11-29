import { spawn, ChildProcess, exec } from "node:child_process";
import { resolve, dirname } from "node:path";
import { mkdirSync } from "node:fs";
import { createServer } from "node:net";
import { request as httpRequest } from "node:http";
import { watch } from "chokidar";
import type { Plugin, ViteDevServer } from "vite";
import type { IncomingMessage, ServerResponse } from "node:http";
import pc from "picocolors";
import { createClient } from "@tygor/client";
import { registry as devtoolsRegistry } from "./devtools/manifest";

/** Tygor manifest registry type */
export interface TygorRegistry {
  metadata: Record<string, { path: string; [key: string]: unknown }>;
}

export interface TygorDevOptions {
  /** Command to run before starting the server (e.g., codegen) */
  prebuild?: string | string[];
  /** Build command (e.g., "go build -o ./tmp/server ."). If provided, build errors are distinguished from runtime errors */
  build?: string | string[];
  /** Path to build output file - parent directory will be created automatically */
  buildOutput?: string;
  /** Function that returns the command to start the server */
  start: (port: number) => {
    cmd: string | string[];
    env?: Record<string, string>;
    cwd?: string;
  };
  /** Tygor manifest registry - proxy paths are derived automatically */
  manifest?: TygorRegistry;
  /** Glob patterns to watch (default: ['**\/*.go']) */
  watch?: string[];
  /** Glob patterns to ignore (default: ['node_modules', '.git', 'tmp']) */
  ignore?: string[];
  /** Health check endpoint - set to false to use TCP probe (default: false) */
  health?: string | false;
  /** Starting port to search from (default: 8080) */
  port?: number;
  /** Working directory for Go commands and file watcher (default: process.cwd()) */
  workdir?: string;
  /** Proxy path prefixes to route to Go server (auto-derived if manifest provided) */
  proxy?: string[];
}

interface ServerState {
  process: ChildProcess | null;
  port: number;
  ready: boolean;
}

const DEFAULT_OPTIONS = {
  watch: ["**/*.go"],
  ignore: ["node_modules", ".git", "tmp", "dist"],
  health: false as const,
  port: 8080,
};

/** Find an available port starting from the given port */
async function findPort(startPort: number): Promise<number> {
  return new Promise((resolve) => {
    const server = createServer();
    server.listen(startPort, () => {
      server.close(() => resolve(startPort));
    });
    server.on("error", () => {
      resolve(findPort(startPort + 1));
    });
  });
}

export function tygorDev(options: TygorDevOptions): Plugin {
  const opts = { ...DEFAULT_OPTIONS, ...options };
  const workdir = resolve(process.cwd(), opts.workdir ?? ".");

  let currentServer: ServerState = { process: null, port: opts.port, ready: false };
  let nextServer: ServerState | null = null;
  let buildError: string | null = null;
  let errorPhase: "prebuild" | "build" | "runtime" | null = null;
  let errorCommand: string | null = null;
  let currentPhase: "idle" | "prebuild" | "building" | "starting" = "idle";
  let isReloading = false;

  const log = (msg: string) => console.log(pc.cyan("[tygor]"), msg);
  const logError = (msg: string) => console.log(pc.cyan("[tygor]"), pc.red(msg));

  async function runPrebuild(): Promise<boolean> {
    if (!opts.prebuild) return true;

    const cmd = Array.isArray(opts.prebuild) ? opts.prebuild.join(" && ") : opts.prebuild;
    log(`Running prebuild: ${cmd}`);

    return new Promise((resolve) => {
      exec(cmd, { cwd: workdir }, (error, stdout, stderr) => {
        if (error) {
          buildError = stderr || error.message;
          errorPhase = "prebuild";
          errorCommand = cmd;
          logError(`Prebuild failed:\n${buildError}`);
          resolve(false);
        } else {
          if (stdout.trim()) console.log(stdout);
          resolve(true);
        }
      });
    });
  }

  async function runBuild(): Promise<boolean> {
    if (!opts.build) {
      log("No build command configured, skipping build step");
      return true;
    }

    // Ensure output directory exists
    if (opts.buildOutput) {
      const outDir = dirname(resolve(workdir, opts.buildOutput));
      log(`Creating output directory: ${outDir}`);
      mkdirSync(outDir, { recursive: true });
    }

    const cmd = Array.isArray(opts.build) ? opts.build.join(" && ") : opts.build;
    log(`Building: ${cmd}`);

    return new Promise((resolve) => {
      exec(cmd, { cwd: workdir }, (error, stdout, stderr) => {
        if (error) {
          buildError = stderr || error.message;
          errorPhase = "build";
          errorCommand = cmd;
          logError(`Build failed:\n${buildError}`);
          resolve(false);
        } else {
          if (stdout.trim()) console.log(stdout);
          resolve(true);
        }
      });
    });
  }

  async function checkHealth(port: number): Promise<boolean> {
    const url = opts.health
      ? `http://localhost:${port}${opts.health}`
      : `http://localhost:${port}/`;

    try {
      const res = await fetch(url);
      log(`Health check ${port}: ${res.status}`);
      // Any response means server is up (even 404)
      return true;
    } catch (e) {
      log(`Health check ${port}: ${e instanceof Error ? e.message : 'failed'}`);
      return false;
    }
  }

  function startServer(port: number, retries = 3): Promise<ServerState> {
    return new Promise((resolve) => {
      const config = opts.start(port);
      const cmdArray = Array.isArray(config.cmd) ? config.cmd : config.cmd.split(" ");
      const [command, ...args] = cmdArray;

      const env = { ...process.env, ...config.env };
      const spawnCwd = config.cwd ?? workdir;

      log(`Starting server on port ${port}`);

      let proc;
      try {
        proc = spawn(command, args, {
          cwd: spawnCwd,
          env,
          stdio: ["ignore", "pipe", "pipe"],
        });
      } catch (err: unknown) {
        const error = err as NodeJS.ErrnoException;
        // ETXTBSY = binary still being written, retry after delay
        if (error.code === "ETXTBSY" && retries > 0) {
          log(`Binary busy, retrying in 200ms (${retries} retries left)`);
          setTimeout(() => {
            startServer(port, retries - 1).then(resolve);
          }, 200);
          return;
        }
        logError(`Failed to spawn: ${error.message}`);
        resolve({ process: null, port, ready: false });
        return;
      }

      let stderr = "";
      let resolved = false;

      proc.stdout?.on("data", (data) => {
        process.stdout.write(pc.dim(data.toString()));
      });

      proc.stderr?.on("data", (data) => {
        stderr += data.toString();
        process.stderr.write(pc.dim(data.toString()));
      });

      proc.on("error", (err) => {
        if (!resolved) {
          resolved = true;
          buildError = err.message;
          errorPhase = "runtime";
          errorCommand = cmdArray.join(" ");
          logError(`Failed to start: ${err.message}`);
          resolve({ process: null, port, ready: false });
        }
      });

      proc.on("exit", (code) => {
        if (!resolved && code !== 0 && code !== null) {
          resolved = true;
          buildError = stderr || `Process exited with code ${code}`;
          errorPhase = "runtime";
          errorCommand = cmdArray.join(" ");
          logError(`Server exited with code ${code}`);
          resolve({ process: null, port, ready: false });
        }
      });

      // Wait for server to be ready - require 2 consecutive successful health checks
      const maxAttempts = 100; // 10 seconds
      let attempts = 0;
      let consecutiveSuccess = 0;

      const pollHealth = async () => {
        attempts++;
        if (resolved) return;

        const healthy = await checkHealth(port);
        if (healthy) {
          consecutiveSuccess++;
          if (consecutiveSuccess >= 2) {
            resolved = true;
            log(`Server ready on port ${port}`);
            buildError = null;
            errorPhase = null;
            errorCommand = null;
            resolve({ process: proc, port, ready: true });
            return;
          }
        } else {
          consecutiveSuccess = 0;
        }

        if (proc.exitCode !== null) {
          // Process exited - error already handled
          if (!resolved) {
            resolved = true;
            resolve({ process: null, port, ready: false });
          }
        } else if (attempts < maxAttempts) {
          setTimeout(pollHealth, 100);
        } else {
          resolved = true;
          logError(`Health check timed out after ${attempts * 100}ms`);
          resolve({ process: proc, port, ready: false });
        }
      };

      // Give the process a moment to start
      setTimeout(pollHealth, 200);
    });
  }

  function killServer(server: ServerState) {
    if (server.process && server.process.exitCode === null) {
      log(`Stopping server on port ${server.port}`);
      server.process.kill("SIGTERM");
    }
  }

  // Start server with port retry logic (handles race between findPort and actual bind)
  async function startServerWithRetry(skipPort?: number): Promise<ServerState> {
    const maxAttempts = 5;
    let port = opts.port;

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      port = await findPort(port);
      if (skipPort && port === skipPort) {
        port = await findPort(port + 1);
      }
      const server = await startServer(port);
      if (server.ready) return server;

      // Port might have been grabbed between findPort and bind - try next
      port++;
      if (attempt < maxAttempts - 1) {
        log(`Port ${server.port} unavailable, trying next...`);
      }
    }
    return { process: null, port, ready: false };
  }

  async function reload() {
    if (isReloading) return;
    isReloading = true;

    log("Detected changes, reloading...");

    // Run prebuild
    const prebuildOk = await runPrebuild();
    if (!prebuildOk) {
      isReloading = false;
      return;
    }

    // Run build (separate from start so we can distinguish build vs runtime errors)
    const buildOk = await runBuild();
    if (!buildOk) {
      isReloading = false;
      return;
    }

    // Start new server on a different port (skip current server's port)
    nextServer = await startServerWithRetry(currentServer.port);

    if (nextServer.ready) {
      // Swap servers - update currentServer first so proxy routes to new server
      const oldServer = currentServer;
      log(`Swapping: ${oldServer.port} -> ${nextServer.port}`);
      currentServer = nextServer;
      nextServer = null;
      buildError = null;

      log(pc.green(`Switched to port ${currentServer.port}`));

      // Give the proxy time to route to new server before killing old one
      setTimeout(() => {
        log(`Killing old server on port ${oldServer.port}`);
        killServer(oldServer);
      }, 500);
    } else {
      // Keep old server, clean up failed new one
      if (nextServer.process) {
        killServer(nextServer);
      }
      nextServer = null;
    }

    isReloading = false;
  }

  // Debounce reload
  let reloadTimeout: ReturnType<typeof setTimeout> | null = null;
  function scheduleReload() {
    if (reloadTimeout) clearTimeout(reloadTimeout);
    reloadTimeout = setTimeout(reload, 300);
  }

  // Derive proxy paths from manifest or use explicit proxy option
  let proxyPaths = opts.proxy ?? [];
  if (opts.manifest && proxyPaths.length === 0) {
    const services = new Set<string>();
    for (const meta of Object.values(opts.manifest.metadata)) {
      const service = meta.path.split("/")[1];
      if (service) services.add(`/${service}`);
    }
    proxyPaths = [...services];
  }

  // Call Devtools.Ping for heartbeat using generated client
  async function heartbeat(): Promise<boolean> {
    if (!currentServer.ready) return false;

    try {
      const client = createClient(devtoolsRegistry, {
        baseUrl: `http://localhost:${currentServer.port}`,
      });
      const res = await client.Devtools.Ping({});
      return res.ok;
    } catch {
      return false;
    }
  }

  // Simple proxy function using Node's http module
  function proxyRequest(req: IncomingMessage, res: ServerResponse) {
    const port = currentServer.port;
    log(`Proxying ${req.url} -> localhost:${port}`);

    const proxyReq = httpRequest(
      {
        hostname: "localhost",
        port,
        path: req.url,
        method: req.method,
        headers: req.headers,
      },
      (proxyRes) => {
        res.writeHead(proxyRes.statusCode ?? 500, proxyRes.headers);
        proxyRes.pipe(res);
      }
    );

    proxyReq.on("error", () => {
      res.writeHead(503, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: { code: "server_unavailable", message: "Go server is starting..." } }));
    });

    req.pipe(proxyReq);
  }

  return {
    name: "tygor-dev",
    apply: "serve",

    async configureServer(server: ViteDevServer) {
      // Status endpoint for polling
      server.middlewares.use((req, res, next) => {
        // Handle tygor status endpoint
        if (req.url === "/__tygor/status") {
          (async () => {
            res.writeHead(200, { "Content-Type": "application/json" });

            let status;
            if (buildError) {
              status = { status: "error", error: buildError, phase: errorPhase ?? "build", command: errorCommand, cwd: workdir };
            } else if (isReloading) {
              status = { status: "reloading" };
            } else if (!currentServer.ready) {
              status = { status: "starting" };
            } else {
              // Get status from Go server (includes services for discovery)
              try {
                const client = createClient(devtoolsRegistry, {
                  baseUrl: `http://localhost:${currentServer.port}`,
                });
                const serverStatus = await client.Devtools.Status({});
                status = { status: "ok", port: serverStatus.port, services: serverStatus.services };
              } catch {
                status = { status: "disconnected" };
              }
            }
            res.end(JSON.stringify(status));
          })();
          return;
        }

        // Proxy API requests to Go server
        if (proxyPaths.some((prefix) => req.url?.startsWith(prefix))) {
          proxyRequest(req, res as ServerResponse);
          return;
        }

        next();
      });

      // Start watcher
      log(`Watching ${opts.watch!.join(", ")} in ${workdir}`);
      const watcher = watch(opts.watch!, {
        cwd: workdir,
        ignored: opts.ignore,
        ignoreInitial: true,
      });

      watcher.on("change", (path) => {
        log(`Changed: ${path}`);
        scheduleReload();
      });

      watcher.on("add", (path) => {
        log(`Added: ${path}`);
        scheduleReload();
      });

      watcher.on("unlink", (path) => {
        log(`Removed: ${path}`);
        scheduleReload();
      });

      // Initial build and server start
      const buildOk = await runBuild();
      if (buildOk) {
        currentServer = await startServerWithRetry();
        if (!currentServer.ready) {
          logError("Server start failed - fix errors and save to retry");
        }
      } else {
        logError("Build failed - fix errors and save to retry");
      }

      // Watchdog: continuously ping server and restart if unresponsive
      let watchdogInterval: ReturnType<typeof setInterval> | null = null;
      let consecutiveFailures = 0;
      const FAILURE_THRESHOLD = 3;

      const watchdog = async () => {
        if (isReloading || !currentServer.ready) return;

        const alive = await heartbeat();
        if (alive) {
          consecutiveFailures = 0;
        } else {
          consecutiveFailures++;
          log(`Heartbeat failed (${consecutiveFailures}/${FAILURE_THRESHOLD})`);

          if (consecutiveFailures >= FAILURE_THRESHOLD) {
            logError("Server unresponsive, restarting...");
            consecutiveFailures = 0;
            killServer(currentServer);
            currentServer = { process: null, port: currentServer.port, ready: false };

            // Restart without rebuild (server crashed, not code change)
            currentServer = await startServerWithRetry();
            if (currentServer.ready) {
              log(pc.green(`Server restarted on port ${currentServer.port}`));
            } else {
              logError("Restart failed - waiting for code change");
            }
          }
        }
      };

      watchdogInterval = setInterval(watchdog, 2000);

      // Cleanup function
      let cleanedUp = false;
      const cleanup = () => {
        if (cleanedUp) return;
        cleanedUp = true;
        log("Shutting down...");
        if (watchdogInterval) clearInterval(watchdogInterval);
        watcher.close();
        killServer(currentServer);
        if (nextServer) killServer(nextServer);
      };

      // Cleanup when Vite's server closes
      server.httpServer?.on("close", cleanup);

      // Also cleanup on process exit (handles SIGTERM, SIGINT, etc.)
      process.on("exit", cleanup);
    },

    transformIndexHtml(html) {
      // Inject client script
      const clientScript = `
<script type="module">
(function() {
  let overlay = null;
  let dismissed = false;
  let lastError = null;

  function createOverlay(error, phase, command, cwd) {
    if (dismissed && error === lastError) return;
    if (error !== lastError) dismissed = false;
    lastError = error;
    if (dismissed) return;

    const phaseLabels = { prebuild: 'Prebuild', build: 'Build', runtime: 'Runtime' };
    const title = 'tygor: Go ' + (phaseLabels[phase] || 'Build') + ' Error';

    // Format for copying - terse but complete context
    const copyText = (command ? cwd + '$ ' + command + '\\n\\n' : '') + error.trim();
    if (overlay) overlay.remove();
    overlay = document.createElement('div');
    overlay.id = 'tygor-error-overlay';
    overlay.innerHTML = \`
      <style>
        #tygor-error-overlay {
          position: fixed; top: 1rem; right: 1rem; z-index: 99999;
          max-width: 520px; max-height: 60vh;
          background: #FFF8F0; color: #222; border-radius: 3px;
          font-family: ui-monospace, 'SF Mono', Consolas, monospace;
          box-shadow: 0 4px 24px rgba(0,0,0,0.12);
          overflow: hidden; display: flex; flex-direction: column;
          border: 1px solid rgba(255,172,66,0.3);
        }
        #tygor-error-overlay .header {
          display: flex; align-items: center; justify-content: space-between;
          padding: 6px 10px;
          background: rgba(255,172,66,0.1);
          border-bottom: 1px solid rgba(255,172,66,0.2);
          font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        #tygor-error-overlay h2 {
          color: #b35309; margin: 0; font-size: 11px; font-weight: 600;
          display: flex; align-items: center; gap: 6px;
        }
        #tygor-error-overlay .actions { display: flex; gap: 4px; }
        #tygor-error-overlay .btn {
          background: none; border: 1px solid rgba(255,172,66,0.3); color: #888;
          font-size: 11px; cursor: pointer; height: 20px; padding: 0 6px;
          border-radius: 2px; display: flex; align-items: center; justify-content: center;
        }
        #tygor-error-overlay .btn:hover { border-color: #b35309; color: #b35309; }
        #tygor-error-overlay .btn.copied { border-color: #16a34a; color: #16a34a; }
        #tygor-error-overlay .body { padding: 12px; overflow: auto; }
        #tygor-error-overlay pre {
          margin: 0; white-space: pre-wrap; font-size: 11px;
          line-height: 1.6; color: #555;
        }
        #tygor-error-overlay .hint {
          color: #888; margin: 10px 0 0; font-size: 10px;
          font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        #tygor-error-overlay .command {
          color: #666; font-size: 10px; margin: 0 0 8px;
        }
      </style>
      <div class="header">
        <h2>\${title}</h2>
        <div class="actions">
          <button class="btn copy" title="Copy error">Copy</button>
          <button class="btn close" title="Dismiss">&times;</button>
        </div>
      </div>
      <div class="body">
        \${command ? '<p class="command">$ ' + command.replace(/</g, '&lt;') + '</p>' : ''}
        <pre>\${error.replace(/</g, '&lt;')}</pre>
        <p class="hint">Fix the error and save — auto-reloads when fixed.</p>
      </div>
    \`;
    overlay.querySelector('.copy').addEventListener('click', (e) => {
      navigator.clipboard.writeText(copyText);
      e.target.textContent = 'Copied';
      e.target.classList.add('copied');
      setTimeout(() => {
        e.target.textContent = 'Copy';
        e.target.classList.remove('copied');
      }, 1500);
    });
    overlay.querySelector('.close').addEventListener('click', () => {
      dismissed = true;
      overlay.remove();
      overlay = null;
    });
    document.body.appendChild(overlay);
  }

  function removeOverlay() {
    if (overlay) { overlay.remove(); overlay = null; }
  }

  let disconnectedSince = null;

  function formatDuration(ms) {
    const seconds = Math.floor(ms / 1000);
    if (seconds < 60) return seconds + 's';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return minutes + 'm';
    return Math.floor(minutes / 60) + 'h';
  }

  function showStatus(message, showDuration) {
    // Don't replace error overlay with status message
    if (overlay && overlay.id === 'tygor-error-overlay') return;
    if (overlay) overlay.remove();

    let durationText = '';
    if (showDuration && disconnectedSince) {
      durationText = ' (' + formatDuration(Date.now() - disconnectedSince) + ')';
    }

    overlay = document.createElement('div');
    overlay.id = 'tygor-status-overlay';
    overlay.innerHTML = \`
      <style>
        #tygor-status-overlay {
          position: fixed; top: 1rem; right: 1rem; z-index: 99999;
          padding: 8px 14px; background: #FFF8F0; color: #b35309;
          border-radius: 3px; font-size: 12px;
          font-family: system-ui, -apple-system, sans-serif;
          box-shadow: 0 2px 8px rgba(0,0,0,0.1);
          border: 1px solid rgba(255,172,66,0.3);
        }
        #tygor-status-overlay .prefix { font-weight: 600; }
        #tygor-status-overlay .duration { opacity: 0.7; }
      </style>
      <span class="prefix">[tygor]</span> \${message}<span class="duration">\${durationText}</span>
    \`;
    document.body.appendChild(overlay);
  }

  async function poll() {
    try {
      const res = await fetch('/__tygor/status');
      const data = await res.json();
      if (data.status === 'error') {
        disconnectedSince = null;
        createOverlay(data.error, data.phase, data.command, data.cwd);
      } else if (data.status === 'reloading') {
        if (!disconnectedSince) disconnectedSince = Date.now();
        showStatus('Reloading Go server...', false);
      } else if (data.status === 'starting') {
        if (!disconnectedSince) disconnectedSince = Date.now();
        showStatus('Starting Go server...', false);
      } else if (data.status === 'disconnected') {
        if (!disconnectedSince) disconnectedSince = Date.now();
        showStatus('Go server disconnected', true);
      } else if (data.status === 'ok') {
        lastError = null;
        dismissed = false;
        disconnectedSince = null;
        removeOverlay();
      }
    } catch {
      if (!disconnectedSince) disconnectedSince = Date.now();
      showStatus('Dev server disconnected', true);
    }
    setTimeout(poll, 1000);
  }
  poll();

})();
</script>`;
      return html.replace("</body>", clientScript + "</body>");
    },
  };
}

export default tygorDev;
