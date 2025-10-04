<script setup lang="ts">
import { onMounted, ref } from 'vue'
// import UserStats from '@/components/UserStats.vue'
import { api } from '@/lib/api'
import type { User } from '@/types'
// import type { Stat } from '@/types'

const whoami = ref<User | null>(null)
const stats = ref()

async function fetchResourceStats() {
  api
    .get('/resources')
    .then((res) => {
      const data = res.data
      stats.value = [
        {
          item: 'Core',
          icon: 'heroicons-solid:chip',
          total: (data.allocated_cores / data.max_cores) * 100,
          active: (data.active_vms_cores / data.max_cores) * 100,
          max: data.max_cores,
          allocated: data.allocated_cores,
          color: 'primary',
        },
        {
          item: 'Nets',
          icon: 'ph:network',
          total: (data.allocated_nets / data.max_nets) * 100,
          active: 0,
          max: data.max_nets,
          allocated: data.allocated_nets,
          color: 'warning',
        },
        {
          item: 'RAM',
          icon: 'fluent:ram-20-regular',
          total: (data.allocated_ram / data.max_ram) * 100,
          active: (data.active_vms_ram / data.max_ram) * 100,
          max: data.max_ram / 1024,
          allocated: data.allocated_ram / 1024,
          color: 'success',
        },
        {
          item: 'Disk',
          icon: 'mingcute:storage-line',
          total: (data.allocated_disk / data.max_disk) * 100,
          active: (data.active_vms_disk / data.max_disk) * 100,
          max: data.max_disk,
          allocated: data.allocated_disk,
          color: 'accent',
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
  <div class="h-full overflow-auto">
    <h1 class="my-3 text-3xl font-bold">Hi {{ whoami?.username }}!</h1>
    <h1 class="my-2 text-xl font-semibold">Usage of your resources</h1>
    <div class="flex flex-wrap justify-around gap-4">
      <!-- <UserStats v-for="stat in stats" :key="stat.item" :stat="stat" /> -->
    </div>
  </div>
</template>
