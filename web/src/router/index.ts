import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/features/auth/store'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomeView.vue'),
    },
    {
      path: '/plans',
      name: 'plans',
      component: () => import('@/views/PlansView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/idea',
      redirect: '/plans',
    },
    {
      path: '/plans/:id',
      name: 'plan-detail',
      component: () => import('@/views/PlanDetailView.vue'),
      meta: { requiresAuth: true },
    },
    { path: '/admin', redirect: '/admin/users' },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('@/views/AdminUsersView.vue'),
      meta: { requiresOwner: true },
    },
    {
      path: '/admin/plans',
      name: 'admin-plans',
      component: () => import('@/views/AdminPlansView.vue'),
      meta: { requiresOwner: true },
    },
    {
      path: '/admin/settings',
      name: 'admin-settings',
      component: () => import('@/views/AdminSettingsView.vue'),
      meta: { requiresOwner: true },
    },
    {
      path: '/admin/storage',
      name: 'admin-storage',
      component: () => import('@/views/AdminStorageView.vue'),
      meta: { requiresOwner: true },
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.initialize()
  if (to.meta.requiresOwner) {
    if (!auth.user) {
      window.location.href = '/api/v1/auth/login'
      return false
    }
    if (auth.user.role !== 'owner') return { path: '/plans' }
  }
  if (to.meta.requiresAuth && !auth.user) {
    window.location.href = '/api/v1/auth/login'
    return false
  }
})

export default router
