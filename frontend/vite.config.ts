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
          'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NTg4NTM5MjEsImlhdCI6MTc1ODgxMDcyMSwidXNlcl9pZCI6NH0.2z23qJiZGLOJdaKjeP1gmlX9FYBBQ07YqBYQBCsMBjc'
        }
      }
    }
  }
})