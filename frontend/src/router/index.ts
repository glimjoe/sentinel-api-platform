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

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.public) return true
  if (!auth.isAuthenticated) return { name: 'login', query: { redirect: to.fullPath } }
  return true
})

export default router
