import axios from 'axios'
import router from '@/router'

import { toast } from '@/composables/useToast'

const api = axios.create({
  baseURL: '/api',
})

// Login function
async function login(username: string, password: string, realm: number): Promise<string> {
  try {
    const response = await api.post('/login', {
      username: username,
      password: password,
      realm: realm,
    })

    // Extract token from Authorization header
    const token = response.headers.authorization?.replace('Bearer ', '')

    if (token) {
      // Store token in localStorage
      localStorage.setItem('jwt_token', token)
    }

    return response.data
  } catch (error) {
    throw error
  }
}

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('jwt_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  },
)

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status
    if (status === 401) {
      // Token expired or invalid
      localStorage.removeItem('jwt_token')

      // Redirect to login page
      if (router.currentRoute.value.path !== '/login') {
        const next = router.currentRoute.value.fullPath
        if (next === '/login' || next === '/') {
          await router.push('/login')
        } else {
          await router.push({ path: '/login', query: { next } })
        }
      }
    } else if (status >= 500 || status === 403) {
      await router.push(`/error/${status}`)
    } else if (status !== 400 && status !== 409) {
      let message = error.message
      message += ': ' + error.response?.data || 'An error occurred'
      toast.error(message)
    }
    return Promise.reject(error)
  },
)

export { api, login }
