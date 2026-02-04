import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: '/ui/',
  build: {
    outDir: 'dist',
    emptyDirOnStart: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
})
