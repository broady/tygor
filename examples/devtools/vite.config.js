import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tygorDev } from "@tygor/vite-plugin";

export default defineConfig({
  plugins: [
    react(),
    tygorDev({
      prebuild: "go run . -gen -out ./src/rpc",
      build: "go build -o ./tmp/server .",
      buildOutput: "./tmp/server",
      start: (port) => ({
        cmd: ["./tmp/server"],
        env: { PORT: String(port) },
      }),
      rpcDir: "./src/rpc",
    }),
  ],
  optimizeDeps: {
    // Exclude from pre-bundling so local file: linked changes are picked up immediately.
    // This can be removed when using a published @tygor/client from npm.
    exclude: ["@tygor/client"],
  },
});
