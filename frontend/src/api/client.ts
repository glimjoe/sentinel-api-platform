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

let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (err: unknown) => void }> = []

function processQueue(token: string | null) {
  failedQueue.forEach(({ resolve, reject }) => {
    token ? resolve(token) : reject(new Error('refresh failed'))
  })
  failedQueue = []
}

// Response interceptor: unwrap envelope, auto-refresh on 401
api.interceptors.response.use(
  (resp) => resp.data,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      const rt = localStorage.getItem('refresh_token')
      if (rt) {
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({
              resolve: (token: string) => {
                originalRequest.headers.Authorization = `Bearer ${token}`
                resolve(api(originalRequest))
              },
              reject,
            })
          })
        }
        originalRequest._retry = true
        isRefreshing = true
        try {
          const resp = await axios.post('/api/v1/auth/refresh', { refresh_token: rt })
          const { access_token, refresh_token } = resp.data.data || resp.data
          if (access_token) localStorage.setItem('access_token', access_token)
          if (refresh_token) localStorage.setItem('refresh_token', refresh_token)
          processQueue(access_token)
          originalRequest.headers.Authorization = `Bearer ${access_token}`
          return api(originalRequest)
        } catch {
          processQueue(null)
          localStorage.removeItem('access_token')
          localStorage.removeItem('refresh_token')
          if (window.location.pathname !== '/login') {
            window.location.href = '/login'
          }
        } finally {
          isRefreshing = false
        }
      } else {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
      }
    }
    const message = error.response?.data?.error ?? error.message
    return Promise.reject(new Error(message))
  },
)
