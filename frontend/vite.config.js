import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // REST API -> Go backend
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // WebSocket live updates -> Go backend
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})