import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 接口代理目标，由 make page:dev:{dev|qa|pro} 注入；默认指向 qa
const apiTarget = process.env.API_TARGET || 'http://192.168.0.25:8090'

export default defineConfig({
  plugins: [react()],
  assetsInclude: ['**/*.wasm'],
  optimizeDeps: {
    exclude: ['@imagemagick/magick-wasm'],
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': { target: apiTarget, changeOrigin: true },
      '/uploads': { target: apiTarget, changeOrigin: true }
    }
  }
})
