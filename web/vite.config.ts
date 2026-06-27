import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The production build is embedded into the Go binary, so the output goes
// straight into the embed directory. A relative base keeps asset URLs working
// regardless of the host path.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: './',
  build: {
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
  },
  server: {
    // Local dev: proxy API + capture + SSE to the Go backend.
    proxy: {
      '/api': { target: 'http://localhost:8084', changeOrigin: true },
      '/health': 'http://localhost:8084',
    },
  },
})
