import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', component: () => import('../views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('../layouts/AdminLayout.vue'),
    children: [
      { path: '', component: () => import('../views/Dashboard.vue'), meta: { perm: 'dashboard:summary' } },
      { path: 'certificates', component: () => import('../views/Certificates.vue'), meta: { perm: 'certificate:list' } },
      { path: 'datasources', component: () => import('../views/Datasources.vue'), meta: { perm: 'datasource:list' } },
      { path: 'pipelines', component: () => import('../views/Pipelines.vue'), meta: { perm: 'etlJob:list' } },
      { path: 'tasks', component: () => import('../views/Tasks.vue'), meta: { perm: 'task:list' } },
      { path: 'schedules', component: () => import('../views/Schedules.vue'), meta: { perm: 'schedule:list' } },
      { path: 'users', component: () => import('../views/Users.vue'), meta: { perm: 'user:list' } },
      { path: 'roles', component: () => import('../views/Roles.vue'), meta: { perm: 'role:list' } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.public) {
    if (auth.token && to.path === '/login') return '/'
    return true
  }
  if (!auth.token) return '/login'
  if (!auth.profile) {
    try {
      await auth.loadProfile()
    } catch {
      auth.logout()
      return '/login'
    }
  }
  if (to.meta.perm && !auth.can(to.meta.perm)) return '/'
  return true
})

export default router
