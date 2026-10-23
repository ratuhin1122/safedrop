import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vitest/config";
import { loadEnv } from "vite";

export default defineConfig(({ mode }) => {
  const { API_PROXY_TARGET } = loadEnv(mode, process.cwd(), "");
  const proxy = API_PROXY_TARGET ? Object.fromEntries(["/v1", "/s"].map((path) => [path, {
    target: API_PROXY_TARGET, changeOrigin: true,
  }])) : undefined;
  return {
    plugins: [react(), tailwindcss()],
    resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
    test: { environment: "jsdom", setupFiles: ["./src/test-setup.ts"] },
    server: { proxy },
  };
});
