import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'

// ADR-0008: Cookie-based auth — withCredentials sends httpOnly cookies
// (sent_access, sent_refresh) automatically. CSRF token is read from the
// readable sent_csrf cookie and sent as X-CSRF-Token header.
export const api: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
})

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)sent_csrf=([^;]*)/)
  return match ? match[1] : ''
}

// Request interceptor: attach CSRF token for state-changing methods.
api.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
  const method = (cfg.method || 'get').toLowerCase()
  if (method !== 'get' && method !== 'head' && method !== 'options') {
    const csrf = getCSRFToken()
    if (csrf && cfg.headers) {
      cfg.headers['X-CSRF-Token'] = csrf
    }
  }
  return cfg
})

let isRefreshing = false
let failedQueue: Array<{ resolve: (value: unknown) => void; reject: (err: unknown) => void }> = []

function processQueue(err: unknown) {
  failedQueue.forEach(({ resolve, reject }) => {
    err ? reject(err) : resolve(undefined)
  })
  failedQueue = []
}

// Response interceptor: unwrap envelope, auto-refresh on 401.
api.interceptors.response.use(
  (resp) => resp.data,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      // ADR-0008: attempt silent refresh via cookie.
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        }).then(() => api(originalRequest))
      }

      originalRequest._retry = true
      isRefreshing = true
      try {
        await axios.post('/api/v1/auth/refresh', {}, { withCredentials: true })
        processQueue(null)
        return api(originalRequest)
      } catch {
        processQueue(new Error('refresh failed'))
        // Redirect to login if not already there.
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
        return Promise.reject(new Error('Session expired'))
      } finally {
        isRefreshing = false
      }
    }

    const message = error.response?.data?.error ?? error.message
    return Promise.reject(new Error(message))
  },
)
