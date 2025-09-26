import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../view/HomeView.vue'
import LoginView from '../view/LoginView.vue'
import VMView from '../view/VMView.vue'
import AdminView from '../view/AdminView.vue'
import AdminUsersView from '../view/admin/UsersView.vue'
import AdminRealmsView from '../view/admin/RealmsView.vue'
import UserDetailView from '../view/admin/UserDetailView.vue'
import RealmsMultiplexer from '../components/realms/RealmsMultiplexer.vue'
import PortForwardApprovalView from '../view/admin/PortForwardApprovalView.vue'
import NetsView from '../view/NetsView.vue'
import SSHKeysView from '../view/SSHKeysView.vue'
import VPNView from '../view/VPNView.vue'
import InterfacesView from '../view/InterfacesView.vue'
import SettingsView from '../view/SettingsView.vue'

import PortForwardView from '../view/PortForwardView.vue'
import { api } from '../lib/api'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: HomeView, meta: { requiresAuth: true } },
    { path: '/login', component: LoginView },
    { path: '/vm', component: VMView, meta: { requiresAuth: true } },
    { path: '/vm/:vmid/interfaces', component: InterfacesView, meta: { requiresAuth: true } },
    { path: '/net', component: NetsView, meta: { requiresAuth: true } },
    { path: '/ssh-keys', component: SSHKeysView, meta: { requiresAuth: true } },
    { path: '/vpn', component: VPNView, meta: { requiresAuth: true } },

    { path: '/port-forward', component: PortForwardView, meta: { requiresAuth: true } },
    { path: '/settings', component: SettingsView, meta: { requiresAuth: true } },
    {
      path: '/admin',
      meta: { requiresAuth: true, requiresAdmin: true },
      children: [
        { path: '', component: AdminView },
        { path: 'users', component: AdminUsersView },
        { path: 'users/:id', component: UserDetailView },
        { path: 'realms', component: AdminRealmsView },
        { path: 'realms/:id', component: RealmsMultiplexer },
        { path: 'ssh-keys', component: () => import('../view/admin/GlobalSSHKeysView.vue') },
        { path: 'port-forwards', component: PortForwardApprovalView },
      ],
    },
    // Catch-all route per rotte non definite
    { path: '/:pathMatch(.*)*', redirect: '/', meta: { requiresAuth: true } },
  ],
})

// Navigation guard per controllare l'autenticazione
router.beforeEach(async (to, from, next) => {
  const token = localStorage.getItem('jwt_token')
  const isAuthenticated = !!token

  // Se la rotta richiede autenticazione
  if (to.meta.requiresAuth && !isAuthenticated) {
    // Reindirizza al login
    next('/login')
    return
  }

  // Se l'utente è già autenticato e va al login, reindirizza alla home
  if (to.path === '/login' && isAuthenticated) {
    next('/')
    return
  }

  // Se la rotta richiede privilegi admin
  if (to.meta.requiresAdmin && isAuthenticated) {
    try {
      const response = await api.get('/whoami')
      const user = response.data
      
      if (user.role !== 'admin') {
        // Non è admin, reindirizza alla home
        next('/')
        return
      }
    } catch (error) {
      console.error('Errore nel verificare il ruolo utente:', error)
      // In caso di errore, reindirizza al login
      next('/login')
      return
    }
  }
  
  next()
})

export default router
