import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../view/HomeView.vue'
import LoginView from '../view/LoginView.vue'
import VMView from '../view/VMView.vue'
import AdminView from '../view/AdminView.vue'
import AdminUsersView from '../view/admin/UsersView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', component: HomeView },
    { path: '/login', component: LoginView },
    { path: '/vm', component: VMView },
    {
      path: '/admin',
      children: [
        { path: '', component: AdminView },
        { path: 'users', component: AdminUsersView },
      ],
    },
  ],
})

export default router
