/**
 * Phase 5b — Auth store unit tests (ADR-0008 cookie-based auth).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

// Mock the API client so we don't make real HTTP calls.
vi.mock('@/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

import { api } from '@/api/client'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('starts with user = null, not authenticated', () => {
    const store = useAuthStore()
    expect(store.user).toBeNull()
    expect(store.isAuthenticated).toBe(false)
    expect(store.loading).toBe(false)
  })

  it('fetchUser sets user on success', async () => {
    const mockUser = { id: 'u1', email: 'a@b.com', role: 'admin' }
    vi.mocked(api.get).mockResolvedValueOnce({ user: mockUser })

    const store = useAuthStore()
    const ok = await store.fetchUser()

    expect(ok).toBe(true)
    expect(store.user).toEqual(mockUser)
    expect(store.isAuthenticated).toBe(true)
    expect(api.get).toHaveBeenCalledWith('/auth/me')
  })

  it('fetchUser returns false and clears user on failure', async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error('401'))

    const store = useAuthStore()
    store.user = { id: 'x', email: 'x@x.com', role: 'viewer' }
    const ok = await store.fetchUser()

    expect(ok).toBe(false)
    expect(store.user).toBeNull()
  })

  it('register sets user from response', async () => {
    const mockUser = { id: 'u2', email: 'new@b.com', role: 'engineer' }
    vi.mocked(api.post).mockResolvedValueOnce({ user: mockUser })

    const store = useAuthStore()
    await store.register('new@b.com', 'pass123')

    expect(store.user).toEqual(mockUser)
    expect(api.post).toHaveBeenCalledWith('/auth/register', {
      email: 'new@b.com',
      password: 'pass123',
      displayName: undefined,
    })
  })

  it('login sets user from response', async () => {
    const mockUser = { id: 'u3', email: 'login@b.com', role: 'admin' }
    vi.mocked(api.post).mockResolvedValueOnce({ user: mockUser })

    const store = useAuthStore()
    await store.login('login@b.com', 'secret')

    expect(store.user).toEqual(mockUser)
    expect(api.post).toHaveBeenCalledWith('/auth/login', {
      email: 'login@b.com',
      password: 'secret',
    })
  })

  it('logout clears user even if server call fails', async () => {
    vi.mocked(api.post).mockRejectedValueOnce(new Error('500'))

    const store = useAuthStore()
    store.user = { id: 'u4', email: 'bye@b.com', role: 'viewer' }

    await store.logout()

    expect(store.user).toBeNull()
    expect(api.post).toHaveBeenCalledWith('/auth/logout')
  })

  it('logout clears user on success', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({})

    const store = useAuthStore()
    store.user = { id: 'u5', email: 'bye2@b.com', role: 'viewer' }

    await store.logout()

    expect(store.user).toBeNull()
  })
})
