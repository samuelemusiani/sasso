import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../view/HomeView.vue'
import LoginView from '../view/LoginView.vue'
import VMView from '../view/VMView.vue'
import AdminView from '../view/AdminView.vue'
import AdminUsersView from '../view/admin/UsersView.vue'
import AdminRealmsView from '../view/admin/RealmsView.vue'
import RealmsMultiplexer from '../components/realms/RealmsMultiplexer.vue'
import NetsView from '../view/NetsView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: HomeView },
    { path: '/login', component: LoginView },
    { path: '/vm', component: VMView },
    { path: '/net', component: NetsView },
    {
      path: '/admin',
      children: [
        { path: '', component: AdminView },
        { path: 'users', component: AdminUsersView },
        { path: 'realms', component: AdminRealmsView },
        { path: 'realms/:id', component: RealmsMultiplexer },
      ],
    },
  ],
})

export default router
