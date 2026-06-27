import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/login' },
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue'), meta: { public: true } },
  { path: '/dashboard', name: 'dashboard', component: () => import('@/views/DashboardView.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Auth guard: redirect to /login if route requires auth and user is not signed in.
router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.public) return true
  if (!auth.isAuthenticated) return { name: 'login', query: { redirect: to.fullPath } }
  return true
})

export default router
