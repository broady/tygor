import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { tygorDev } from "@tygor/vite-plugin";

export default defineConfig({
  plugins: [
    solid(),
    tygorDev({
      proxyPrefix: "/api",
      // gen: true (default) - runs `tygor gen` to generate TypeScript types
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
