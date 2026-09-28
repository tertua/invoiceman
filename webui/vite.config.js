import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";
import adminDevRewrite from "./scripts/admin-dev-rewrite.mjs";

export default defineConfig({
  plugins: [react(), tailwindcss(), adminDevRewrite()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "http://localhost:5000", changeOrigin: true },
      "/uploads": { target: "http://localhost:5000", changeOrigin: true },
    },
  },
  build: {
    // PDF is intentionally large but loaded only after the user requests a PDF.
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      input: {
        index: path.resolve(import.meta.dirname, "index.html"),
        admin: path.resolve(import.meta.dirname, "admin.html"),
      },
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          if (id.includes("@react-pdf")) return "pdf";
          if (id.includes("recharts") || id.includes("d3-")) return "charts";
          if (id.includes("framer-motion")) return "motion";
          if (
            id.includes("react/") ||
            id.includes("react-dom") ||
            id.includes("react-router") ||
            id.includes("@tanstack") ||
            id.includes("axios")
          ) {
            return "framework";
          }
        },
      },
    },
  },
});
