import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'

export const api: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

// Request interceptor: attach access token if present
api.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('access_token')
  if (token && cfg.headers) {
    cfg.headers.Authorization = `Bearer ${token}`
  }
  return cfg
})

// Response interceptor: unwrap envelope, normalize errors
api.interceptors.response.use(
  (resp) => resp.data,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired or invalid. Drop and bounce to login.
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    // Backend writes `error` (see api/auth.go writeAuthError), not `message`.
    const message = error.response?.data?.error ?? error.message
    return Promise.reject(new Error(message))
  },
)
