import axios from 'axios'

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
  (error) => {
    if (error.response?.status === 401) {
      // Token expired or invalid
      localStorage.removeItem('jwt_token')
      // Redirect to login page
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export { api, login }
