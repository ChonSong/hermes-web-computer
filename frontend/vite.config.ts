import { defineConfig } from "vite"
import { svelte } from "@sveltejs/vite-plugin-svelte"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: {
    proxy: {
      "/ws": {
        target: "ws://localhost:3005",
        ws: true,
      },
      "/api": {
        target: "http://localhost:3005",
        changeOrigin: true,
      },
      "/v1/models": {
        target: "http://localhost:8642",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    target: "esnext",
    rollupOptions: {
      output: {
        manualChunks: {
          // Monaco editor gets its own ~3.7MB chunk (loaded on demand)
          "monaco-editor": ["monaco-editor"],
          // Vendor chunks for better caching
          "svelte-vendor": ["svelte"],
          "xterm": ["@xterm/xterm", "@xterm/addon-fit"],
        },
      },
    },
  },
})
