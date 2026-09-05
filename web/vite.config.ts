import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In dev, proxy /api to the gateway so the frontend can call it same-origin.
export default defineConfig({
  plugins: [react()],
  server: {
    host: true, // bind IPv4 + IPv6 (Firefox resolves localhost to 127.0.0.1)
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": { target: "http://127.0.0.1:8080", changeOrigin: true },
    },
  },
});
