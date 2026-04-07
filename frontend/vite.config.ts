import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // 将 Vue 相关库分离到单独的 chunk
          'vue-vendor': ['vue', 'vue-router', 'vue-i18n', 'pinia'],
          // 将 Naive UI 分离到单独的 chunk
          'naive-ui': ['naive-ui'],
          // 将图标库分离
          'icons': ['@vicons/ionicons5']
        }
      }
    },
    // 提高 chunk 大小警告限制（可选，如果不想看到警告）
    chunkSizeWarningLimit: 1000
  }
})
