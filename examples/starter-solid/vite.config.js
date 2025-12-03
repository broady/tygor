import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { tygorDev } from "@tygor/vite-plugin";

export default defineConfig({
  plugins: [
    solid(),
    tygorDev({
      proxyPrefix: "/api",
      // gen: true (default) - runs `tygor gen` to generate TypeScript types
      build: "go build -o ./.tygor/server .",
      buildOutput: "./.tygor/server",
      start: (port) => ({
        cmd: ["./.tygor/server"],
        env: { PORT: String(port) },
      }),
      rpcDir: "./src/rpc",
    }),
  ],
  // Exclude local packages from Vite's dep optimization cache during development.
  // Without this, changes to @tygor/client require manually clearing node_modules/.vite/
  optimizeDeps: {
    exclude: ["@tygor/client"],
  },
});
