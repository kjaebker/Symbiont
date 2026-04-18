import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    allowedHosts: process.env.VITE_ALLOWED_HOSTS
      ? process.env.VITE_ALLOWED_HOSTS.split(',')
      : [],
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL ?? 'http://localhost:8421',
        changeOrigin: true,
      },
      '/data': {
        target: process.env.VITE_API_URL ?? 'http://localhost:8421',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
  },
})
