import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'

export interface User {
  id: string
  email: string
  displayName?: string
  role: 'admin' | 'engineer' | 'viewer'
}

// ADR-0008: Tokens are httpOnly cookies. The store no longer reads/writes
// localStorage. Auth state is determined by calling /auth/me.
export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isAuthenticated = computed(() => !!user.value)

  // fetchUser is called by the router guard on page load to check
  // whether the user has valid httpOnly cookies.
  async function fetchUser(): Promise<boolean> {
    loading.value = true
    try {
      const resp: any = await api.get('/auth/me')
      user.value = resp.user ?? resp.data?.user ?? null
      return !!user.value
    } catch {
      user.value = null
      return false
    } finally {
      loading.value = false
    }
  }

  async function login(email: string, password: string): Promise<void> {
    const resp: any = await api.post('/auth/login', { email, password })
    // Cookies (sent_access, sent_refresh, sent_csrf) are set by the server.
    user.value = resp.user ?? resp.data?.user ?? null
  }

  async function register(email: string, password: string, displayName?: string): Promise<void> {
    const resp: any = await api.post('/auth/register', {
      email,
      password,
      displayName,
    })
    user.value = resp.user ?? resp.data?.user ?? null
  }

  async function logout(): Promise<void> {
    try {
      await api.post('/auth/logout')
    } catch {
      // Even if the server call fails, clear local state.
    }
    user.value = null
  }

  return { user, loading, isAuthenticated, fetchUser, login, register, logout }
})
