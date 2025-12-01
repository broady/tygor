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
});
