<script setup lang="ts">
import AdminCard from '@/components/AdminCard.vue'
import { getPageIcon } from '@/const'

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
    icon: getPageIcon('groups'),
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
    icon: getPageIcon('ssh-keys'),
    title: 'SSH Keys',
  },
  portForwards: {
    to: '/admin/port-forwards',
    color: 'success',
    icon: getPageIcon('port-forwards'),
    title: 'Port Forwards',
  },
}
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" :icon="getPageIcon('admin')"></IconVue>Admin Panel
      </h1>
      <HelpButton />
    </div>

    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
      <AdminCard v-for="(value, key) in menu" :key="key" :value="value"></AdminCard>
    </div>
  </div>
</template>
