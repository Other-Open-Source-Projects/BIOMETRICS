import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:59013",
      "/health": "http://localhost:59013"
    }
  }
});
