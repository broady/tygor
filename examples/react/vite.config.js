import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tygor } from "@tygor/vite-plugin";

export default defineConfig({
  plugins: [
    react(),
    tygor({
      proxyPrefix: "/api",
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
