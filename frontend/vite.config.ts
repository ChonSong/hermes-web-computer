import { defineConfig } from "vite"
import { svelte } from "@sveltejs/vite-plugin-svelte"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: {
    proxy: {
      "/ws": {
        target: "ws://localhost:3001",
        ws: true,
      },
    },
  },
  build: {
    outDir: "dist",
    target: "esnext",
  },
})
