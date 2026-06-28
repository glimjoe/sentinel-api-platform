import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'

export interface User {
  id: string
  email: string
  displayName?: string
  role: 'admin' | 'engineer' | 'viewer'
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  user: User
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(localStorage.getItem('access_token'))
  const refreshToken = ref<string | null>(localStorage.getItem('refresh_token'))
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => !!accessToken.value)

  function setTokens(at: string, rt: string) {
    accessToken.value = at
    refreshToken.value = rt
    localStorage.setItem('access_token', at)
    localStorage.setItem('refresh_token', rt)
  }

  function clear() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  async function login(email: string, password: string): Promise<void> {
    const resp = await api.post<AuthResponse, AuthResponse>('/auth/login', { email, password })
    setTokens(resp.access_token, resp.refresh_token)
    user.value = resp.user
  }

  async function register(email: string, password: string, displayName?: string): Promise<void> {
    const resp = await api.post<AuthResponse, AuthResponse>('/auth/register', {
      email,
      password,
      displayName,
    })
    setTokens(resp.access_token, resp.refresh_token)
    user.value = resp.user
  }

  function logout() {
    clear()
  }

  async function refresh(): Promise<boolean> {
    if (!refreshToken.value) return false
    try {
      const resp = await api.post<AuthResponse, AuthResponse>('/auth/refresh', { refresh_token: refreshToken.value })
      setTokens(resp.access_token, resp.refresh_token)
      user.value = resp.user
      return true
    } catch {
      clear()
      return false
    }
  }

  return { accessToken, refreshToken, user, isAuthenticated, login, register, logout, refresh }
})
