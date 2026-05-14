import { defineConfig } from "vite"
import { svelte } from "@sveltejs/vite-plugin-svelte"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: {
    proxy: {
      "/ws": {
        target: "ws://localhost:3113",
        ws: true,
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
  },
})
