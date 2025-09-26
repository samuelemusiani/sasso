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
          'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTg5Mjk2NzYsImlhdCI6MTc1ODg4NjQ3NiwidXNlcl9pZCI6M30.VUs6zrvYT-A-CgHJMpx1dAj0hnZCh7PcfdRIg1k2j4g'
        }
      }
    }
  }
})