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
    { path: '/plans', name: 'plans', component: () => import('@/views/PlansView.vue'), meta: { requiresAuth: true } },
    { path: '/admin', name: 'admin', component: () => import('@/views/AdminView.vue'), meta: { requiresOwner: true } },
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
  if (to.meta.requiresOwner && auth.user?.role !== 'owner') return auth.user ? '/' : { path: '/', query: { auth: 'required' } }
  if (to.meta.requiresAuth && !auth.user) { window.location.href = '/api/v1/auth/login'; return false }
})

export default router
