// [snippet:vite-config]

import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { tygor } from "@tygor/vite-plugin";

export default defineConfig({
  plugins: [
    solid(),
    tygor({
      proxyPrefix: "/api",
      // Prebuild: generate Go code from SQL before building
      prebuild: "sqlc generate",
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
  optimizeDeps: {
    exclude: ["@tygor/client"],
  },
});
// [/snippet:vite-config]
