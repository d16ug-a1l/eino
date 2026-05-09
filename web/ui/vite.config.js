import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
  },
  server: {
    port: 3000,
    proxy: {
      '/chat': 'http://localhost:8080',
      '/resume': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    },
  },
})
