import { defineConfig } from 'vite'
import electron from 'vite-plugin-electron'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))

// Build only Electron main+preload. Frontend is built separately.
// electron-builder packs dist/ which contains the electron output,
// and the Go backend (running separately) serves the frontend at localhost:3113.
export default defineConfig({
  plugins: [
    electron([
      {
        entry: resolve(__dirname, 'src/main/index.ts'),
        onstart(options) {
          options.startup()
        },
        vite: {
          build: {
            outDir: 'dist/main',
            rollupOptions: {
              external: ['electron', 'electron-store', 'electron-log']
            }
          }
        }
      },
      {
        entry: resolve(__dirname, 'src/preload/index.ts'),
        onstart(options) {
          options.reload()
        },
        vite: {
          build: {
            outDir: 'dist/preload',
            rollupOptions: {
              external: ['electron']
            }
          }
        }
      }
    ])
  ],
  // Keep root as desktop dir so config can resolve 'vite'
  root: __dirname,
  build: {
    outDir: 'dist/electron',
    emptyOutDir: true
  }
})