import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/dashboard' },
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue'), meta: { public: true } },
  { path: '/dashboard', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
  { path: '/projects', name: 'projects', component: () => import('@/views/ProjectListView.vue') },
  { path: '/projects/:pid', name: 'projectDetail', component: () => import('@/views/ProjectDetailView.vue') },
  { path: '/projects/:pid/console', name: 'mockConsole', component: () => import('@/views/MockConsoleView.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// ADR-0008: Cookie-based auth. On first visit, try /auth/me to restore
// the session from httpOnly cookies. If that fails, redirect to login.
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.public) return true

  // Lazy-fetch user on first protected navigation.
  if (!auth.user) {
    const ok = await auth.fetchUser()
    if (!ok) return { name: 'login', query: { redirect: to.fullPath } }
  }

  return true
})

export default router
