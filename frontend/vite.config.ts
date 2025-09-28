import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    proxy: {
      '/api': {
        target: "https://sasso.mini.students.cs.unibo.it",
        changeOrigin: true,
        headers: {
          'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTkxMzUzNDQsImlhdCI6MTc1OTA5MjE0NCwidXNlcl9pZCI6Mn0.w-U-AkqhCCt-lgpzdpfT0PGmHdhdIB9VDKQVhAmjFsM'
        }
      }
    }
  }
})
