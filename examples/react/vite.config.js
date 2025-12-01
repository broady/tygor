import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { registry } from "./src/rpc/manifest";

// Derive unique service prefixes from the generated manifest
const services = [
  ...new Set(Object.values(registry.metadata).map((m) => m.path.split("/")[1])),
];

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: Object.fromEntries(
      services.map((s) => [
        `/${s}`,
        {
          target: "http://localhost:8080",
          // Suppress noisy errors during Go server startup/restart
          configure: (proxy) => {
            proxy.on("error", (err, req, res) => {
              if (err.code === "ECONNREFUSED") {
                res.writeHead(503);
                res.end("Go server starting...");
              }
            });
          },
        },
      ])
    ),
  },
});
