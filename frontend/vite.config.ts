import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 28031,
    proxy: {
      '/api': {
        target: 'http://localhost:29069',
        changeOrigin: true,
      },
      // 头像等公开静态资源（案件文件不经此入口，统一走 /api 授权下载）
      '/uploads': {
        target: 'http://localhost:29069',
        changeOrigin: true,
      },
    },
  },
})
