import { createRouter, createWebHistory } from 'vue-router'

import HomeView from '../view/HomeView.vue'
import LoginView from '../view/LoginView.vue'
import VMView from '../view/VMView.vue'
import AdminView from '../view/AdminView.vue'
import AdminUsersView from '../view/admin/UsersView.vue'
import AdminRealmsView from '../view/admin/RealmsView.vue'
import UserDetailView from '../view/admin/UserDetailView.vue'
import RealmsMultiplexer from '../components/realms/RealmsMultiplexer.vue'
import NetsView from '../view/NetsView.vue'
import PortForwardsView from '../view/PortForwardsView.vue'
import AdminPortForwardsView from '../view/admin/PortForwardsView.vue'
import SSHKeysView from '../view/SSHKeysView.vue'
import VPNView from '../view/VPNView.vue'
import InterfacesView from '../view/InterfacesView.vue'
import BackupsView from '../view/BackupsView.vue'
import SettingsView from '@/view/SettingsView.vue'
import SidebarView from '@/view/SidebarView.vue'
import GlobalSSHKeysView from '@/view/admin/GlobalSSHKeysView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/login', component: LoginView },
    {
      path: '/',
      component: SidebarView,
      children: [
        { path: '', component: HomeView },
        { path: '/vm', component: VMView },
        { path: '/vm/:vmid/interfaces', component: InterfacesView },
        { path: '/vm/:vmid/backups', component: BackupsView },
        { path: '/net', component: NetsView },
        { path: '/ssh-keys', component: SSHKeysView },
        { path: '/vpn', component: VPNView },
        { path: '/port-forwards', component: PortForwardsView },
        { path: '/settings', component: SettingsView },
        {
          path: '/admin',
          children: [
            { path: '', component: AdminView },
            { path: 'users', component: AdminUsersView },
            { path: 'users/:id', component: UserDetailView },
            { path: 'realms', component: AdminRealmsView },
            { path: 'realms/:id', component: RealmsMultiplexer },
            { path: 'ssh-keys', component: GlobalSSHKeysView },
            { path: 'port-forwards', component: AdminPortForwardsView },
          ],
        },
      ],
    },
  ],
})

export default router
