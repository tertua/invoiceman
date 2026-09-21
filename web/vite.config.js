import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:5000",
        changeOrigin: true,
      },
    },
  },
  build: {
    // PDF is intentionally large but loaded only after the user requests a PDF.
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
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
