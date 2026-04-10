import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import EmptyView from '../view/EmptyView.vue'
import HomeView from '../view/HomeView.vue'
import LoginView from '../view/LoginView.vue'
import VMView from '../view/VMView.vue'
import VMsView from '../view/VMsView.vue'
import AdminView from '../view/AdminView.vue'
import AdminUsersView from '../view/admin/UsersView.vue'
import AdminGroupsView from '../view/admin/GroupsView.vue'
import AdminRealmsView from '../view/admin/RealmsView.vue'
import UserDetailView from '../view/admin/UserDetailView.vue'
import GroupDetailView from '../view/admin/GroupDetailView.vue'
import RealmsMultiplexer from '../components/realms/RealmsMultiplexer.vue'
import NetsView from '../view/NetsView.vue'
import PortForwardsView from '../view/PortForwardsView.vue'
import AdminPortForwardsView from '../view/admin/PortForwardsView.vue'
import SSHKeysView from '../view/SSHKeysView.vue'
import VPNView from '../view/VPNView.vue'
import InterfacesView from '../view/InterfacesView.vue'
import SettingsView from '@/view/SettingsView.vue'
import SidebarView from '@/view/SidebarView.vue'
import ErrorPage from '../view/ErrorPage.vue'
import GlobalSSHKeysView from '@/view/admin/GlobalSSHKeysView.vue'
import TelegramView from '@/view/TelegramView.vue'
import GroupsView from '@/view/GroupsView.vue'
import SingleGroupView from '@/view/SingleGroupView.vue'
import VMInfo from '@/components/vm/VMInfo.vue'
import VMResources from '@/components/vm/VMResources.vue'
import VMInterfaces from '@/components/vm/VMInterfaces.vue'
import VMBackups from '@/components/vm/VMBackups.vue'
import VMBackupsRequests from '@/components/vm/VMBackupsRequests.vue'
import VMBackupsBase from '@/components/vm/VMBackupsBase.vue'
import HelpView from '@/view/HelpView.vue'
import HelpHome from '@/components/help/HelpHome.vue'
import HelpVMs from '@/components/help/HelpVMs.vue'
import HelpVPN from '@/components/help/HelpVPN.vue'
import HelpSSHKeys from '@/components/help/HelpSSHKeys.vue'
import HelpVMBackups from '@/components/help/HelpVMBackups.vue'
import HelpNets from '@/components/help/HelpNets.vue'
import HelpVMInfo from '@/components/help/HelpVMInfo.vue'
import HelpVMResources from '@/components/help/HelpVMResources.vue'
import HelpVMInterfaces from '@/components/help/HelpVMInterfaces.vue'
import HelpInterfaces from '@/components/help/HelpInterfaces.vue'
import HelpSettings from '@/components/help/HelpSettings.vue'
import HelpTelegram from '@/components/help/HelpTelegram.vue'
import HelpPortForwards from '@/components/help/HelpPortForwards.vue'

const routes: RouteRecordRaw[] = [
  { path: '/login', component: LoginView },
  {
    path: '/',
    component: SidebarView,
    redirect: '/home',
    children: [
      { path: '/home', component: HomeView, meta: { helpComponent: HelpHome } },
      {
        path: '/vm',
        component: EmptyView,
        meta: { helpComponent: HelpVMs },
        children: [
          { path: '', name: 'vm-list', component: VMsView },
          {
            path: '/vm/:vmid',
            component: VMView,
            redirect: { name: 'vm-info' },
            children: [
              {
                path: 'info',
                name: 'vm-info',
                component: VMInfo,
                meta: { helpComponent: HelpVMInfo },
              },
              {
                path: 'resources',
                name: 'vm-resources',
                component: VMResources,
                meta: { helpComponent: HelpVMResources },
              },
              {
                path: 'interfaces',
                name: 'vm-interfaces',
                component: VMInterfaces,
                meta: { helpComponent: HelpVMInterfaces },
              },
              {
                path: 'backups',
                name: 'vm-backups',
                component: VMBackupsBase,
                children: [
                  {
                    path: '',
                    name: 'vm-backups-list',
                    component: VMBackups,
                    meta: { helpComponent: HelpVMBackups },
                  },
                  { path: 'requests', name: 'vm-backup-requests', component: VMBackupsRequests },
                ],
              },
            ],
          },
        ],
      },
      { path: '/net', component: NetsView, meta: { helpComponent: HelpNets } },
      { path: '/interfaces', component: InterfacesView, meta: { helpComponent: HelpInterfaces } },
      { path: '/ssh-keys', component: SSHKeysView, meta: { helpComponent: HelpSSHKeys } },
      { path: '/vpn', component: VPNView, meta: { helpComponent: HelpVPN } },
      {
        path: '/port-forwards',
        component: PortForwardsView,
        meta: { helpComponent: HelpPortForwards },
      },
      { path: '/telegram', component: TelegramView, meta: { helpComponent: HelpTelegram } },
      { path: '/settings', component: SettingsView, meta: { helpComponent: HelpSettings } },
      {
        path: '/group',
        children: [
          { path: '', component: GroupsView },
          { path: ':id', component: SingleGroupView },
        ],
      },
      {
        path: '/admin',
        children: [
          { path: '', component: AdminView },
          { path: 'users', component: AdminUsersView },
          { path: 'users/:id', component: UserDetailView },
          { path: 'groups', component: AdminGroupsView },
          { path: 'groups/:id', component: GroupDetailView },
          { path: 'realms', component: AdminRealmsView },
          { path: 'realms/:id', component: RealmsMultiplexer },
          { path: 'ssh-keys', component: GlobalSSHKeysView },
          { path: 'port-forwards', component: AdminPortForwardsView },
        ],
      },
      {
        path: '/help',
        component: HelpView,
        meta: { fullscreen: true },
        children: [
          { path: 'home', component: HelpHome, meta: { title: 'home' } },
          {
            path: 'vm',
            meta: { title: 'vm' },
            children: [
              { path: '', component: HelpVMs, meta: { title: 'vm' } },
              { path: 'info', component: HelpVMInfo, meta: { title: 'info' } },
              { path: 'resources', component: HelpVMResources, meta: { title: 'resources' } },
              { path: 'interfaces', component: HelpVMInterfaces, meta: { title: 'interfaces' } },
              { path: 'backups', component: HelpVMBackups, meta: { title: 'backups' } },
            ],
          },
          { path: 'net', component: HelpNets, meta: { title: 'nets' } },
          { path: 'interfaces', component: HelpInterfaces, meta: { title: 'interfaces' } },
          { path: 'ssh-keys', component: HelpSSHKeys, meta: { title: 'ssh-keys' } },
          { path: 'vpn', component: HelpVPN, meta: { title: 'vpn' } },
          { path: 'port-forwards', component: HelpPortForwards, meta: { title: 'port-forwards' } },
          { path: 'telegram', component: HelpTelegram, meta: { title: 'telegram' } },
          { path: 'settings', component: HelpSettings, meta: { title: 'settings' } },
        ],
      },
    ],
  },
  {
    path: '/error/:code',
    name: 'Error',
    component: ErrorPage,
    props: true, // Pass route params as props
  },
  // 404 - Catch all (must be last!)
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: ErrorPage,
    props: { code: 404 }, // Default to 404
  },
]

export default createRouter({ history: createWebHistory(import.meta.env.BASE_URL), routes })
