import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': resolve(__dirname, 'src') },
  },
  build: {
    outDir: 'dist',
  },
  server: {
    port: 3000,
    proxy: {
      '/api/v1': { target: 'http://localhost:9000', changeOrigin: true },
    },
  },
})
