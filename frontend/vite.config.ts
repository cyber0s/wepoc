import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  esbuild: {
    // 跳过类型检查以加快构建速度
    target: 'es2020'
  },
  build: {
    // 跳过类型检查
    rollupOptions: {
      onwarn(warning, warn) {
        // 忽略类型检查警告
        if (warning.code === 'UNRESOLVED_IMPORT') return
        warn(warning)
      }
    }
  }
})
