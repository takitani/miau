import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { fileURLToPath } from 'url'
import { dirname, resolve } from 'path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
    minify: 'esbuild',
    target: 'esnext'
  },
  resolve: {
    alias: {
      // Alias for old wailsjs bindings path to new v3 bindings
      '$lib/wailsjs/go/desktop/App': resolve(__dirname, 'bindings/github.com/opik/miau/internal/desktop/app.js'),
      '$lib/wailsjs/go/desktop/App.js': resolve(__dirname, 'bindings/github.com/opik/miau/internal/desktop/app.js'),
      '../wailsjs/wailsjs/go/desktop/App.js': resolve(__dirname, 'bindings/github.com/opik/miau/internal/desktop/app.js'),
      './lib/wailsjs/wailsjs/go/desktop/App.js': resolve(__dirname, 'bindings/github.com/opik/miau/internal/desktop/app.js')
    }
  },
  server: {
    strictPort: true
  }
})
