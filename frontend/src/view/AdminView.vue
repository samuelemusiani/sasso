<script setup lang="ts">
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import AdminCard from '@/components/AdminCard.vue'

const menu = {
  users: {
    to: '/admin/users',
    color: 'info',
    icon: 'mdi:account-multiple-outline',
    title: 'Users',
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
  <div class="p-2">
    <AdminBreadcrumbs />
    <!-- Header -->
    <div class="mb-8 px-2">
      <div class="flex items-center gap-3 mb-4 text-3xl font-bold">
        <IconVue icon="material-symbols:admin-panel-settings" class="text-primary" />
        <h1 class="text-base-content">Admin Panel</h1>
      </div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8 px-2">
      <AdminCard v-for="(value, key) in menu" :key="key" :value="value"></AdminCard>
    </div>
  </div>
</template>
