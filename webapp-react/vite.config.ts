import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../webapp',
    emptyOutDir: true, // Удаляем старые файлы при сборке
    rollupOptions: {
      input: path.resolve(__dirname, 'index.html'),
    },
  },
  publicDir: 'public',
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8000', // 8000 для dev, 10000 для prod
        changeOrigin: true,
      },
    },
  },
})

