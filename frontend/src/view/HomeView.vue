<script setup lang="ts">
import { onMounted, ref } from 'vue'
import UserStats from '@/components/UserStats.vue'
import { api } from '@/lib/api'
import type { User } from '@/types'

const whoami = ref<User | null>(null)
const stats = ref()

async function fetchResourceStats() {
  api
    .get('/resources')
    .then((res) => {
      const data = res.data
      stats.value = [
        {
          item: 'CPU',
          icon: 'heroicons-solid:chip',
          active: data.active_vms_cores,
          max: data.max_cores,
          allocated: data.allocated_cores,
          color: 'primary',
        },
        {
          item: 'RAM',
          icon: 'fluent:ram-20-regular',
          active: data.active_vms_ram / 1024,
          max: data.max_ram / 1024,
          allocated: data.allocated_ram / 1024,
          color: 'success',
        },
        {
          item: 'Disk',
          icon: 'mingcute:storage-line',
          active: data.active_vms_disk,
          max: data.max_disk,
          allocated: data.allocated_disk,
          color: 'accent',
        },
        {
          item: 'Net',
          icon: 'ph:network',
          active: 0,
          max: data.max_nets,
          allocated: data.allocated_nets,
          color: 'orange-400',
        },
      ]
    })
    .catch((err) => {
      console.error('Failed to fetch resource stats:', err)
    })
}

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      whoami.value = res.data as User
    })
    .catch((err) => {
      console.error('Failed to fetch whoami:', err)
    })
}

onMounted(() => {
  fetchWhoami()
  fetchResourceStats()
})
</script>

<template>
  <div class="h-full overflow-auto p-4">
    <div class="mb-6">
      <h1 class="my-3 flex items-center gap-3 text-3xl font-bold">Hi {{ whoami?.username }}!</h1>
      <h2 class="text-base-content/80 my-2 text-xl font-semibold">Usage of your resources</h2>
    </div>

    <UserStats v-if="stats" :stats="stats" />

    <div v-else class="flex h-64 items-center justify-center">
      <span class="loading loading-spinner loading-lg text-primary"></span>
    </div>
  </div>
</template>
