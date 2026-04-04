<script setup lang="ts">
import AdminCard from '@/components/AdminCard.vue'

const menu = {
  users: {
    to: '/admin/users',
    color: 'info',
    icon: 'mdi:account-multiple-outline',
    title: 'Users',
  },
  groups: {
    to: '/admin/groups',
    color: 'error',
    icon: 'mdi:account-group-outline',
    title: 'Groups',
  },
  realms: {
    to: '/admin/realms',
    color: 'primary',
    icon: 'mdi:domain',
    title: 'Realms',
  },
  sshKeys: {
    to: '/admin/ssh-keys',
    color: 'warning',
    icon: 'mdi:key-outline',
    title: 'SSH Keys',
  },
  portForwards: {
    to: '/admin/port-forwards',
    color: 'success',
    icon: 'mdi:network-outline',
    title: 'Port Forwards',
  },
}
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" icon="material-symbols:admin-panel-settings"></IconVue>Admin
        Panel
      </h1>
      <HelpButton />
    </div>

    <div class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
      <AdminCard v-for="(value, key) in menu" :key="key" :value="value"></AdminCard>
    </div>
  </div>
</template>
