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
//   server: {
//     proxy: {
//       '/api': {
//         target: "http://localhost:8080",
//         changeOrigin: true
//       }
//     }
//   }
// })

  server: {
    proxy: {
      '/api': {
        target: "https://sasso.mini.students.cs.unibo.it",
        changeOrigin: true,
        headers: {
          'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTg5MTU2NDQsImlhdCI6MTc1ODg3MjQ0NCwidXNlcl9pZCI6NH0.j4HnmAIDA-p4VZhYH4uFQ5UIpXmcc5FktaXWPd0YJDY'
        }
      }
    }
  }
})