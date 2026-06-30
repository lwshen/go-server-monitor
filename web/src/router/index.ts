import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

// Lazy-loaded pages. Real auth guards / WebSocket scope handling land in P5/P6.
const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('@/pages/DashboardPage.vue'),
    meta: { requireAuth: false },
  },
  {
    path: '/server/:id',
    name: 'server-detail',
    component: () => import('@/pages/ServerDetailPage.vue'),
    meta: { requireAuth: false },
  },
  {
    path: '/admin',
    name: 'admin',
    component: () => import('@/pages/AdminPage.vue'),
    meta: { requireAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// TODO(P6): beforeEach guard — redirect to login when meta.requireAuth and no JWT.

export default router
