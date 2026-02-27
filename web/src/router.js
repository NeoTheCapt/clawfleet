import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from './composables/useAuth'

const routes = [
  { path: '/login', name: 'Login', component: () => import('./views/Login.vue') },
  { path: '/im', name: 'IM', component: () => import('./views/IM.vue') },
  { path: '/', name: 'Dashboard', component: () => import('./views/Dashboard.vue'), meta: { auth: true } },
  { path: '/nodes', name: 'Nodes', component: () => import('./views/Nodes.vue'), meta: { auth: true } },
  { path: '/agents', name: 'Agents', component: () => import('./views/Agents.vue'), meta: { auth: true } },
  { path: '/companies', name: 'Companies', component: () => import('./views/Companies.vue'), meta: { auth: true } },
  { path: '/company/:id', name: 'CompanyDetail', component: () => import('./views/company/CompanyDetail.vue'), meta: { auth: true } },
  { path: '/settings', name: 'Settings', component: () => import('./views/Settings.vue'), meta: { auth: true } }
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  if (to.meta.auth && !isAuthenticated()) return { name: 'Login' }
  if (to.name === 'Login' && isAuthenticated()) return { name: 'Dashboard' }
})

export default router
