import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tygorDev } from "../../../vite-plugin/src/index.ts";

export default defineConfig({
  plugins: [
    react(),
    tygorDev({
      prebuild: "go run . -gen -out ./client/src/rpc",
      build: "go build -o ./tmp/server .",
      buildOutput: "./tmp/server",
      start: (port) => ({
        cmd: ["./tmp/server"],
        env: { PORT: String(port) },
      }),
      workdir: "..",
      rpcDir: "./src/rpc",
    }),
  ],
});
