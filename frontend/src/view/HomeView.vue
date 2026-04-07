<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import UserStats from '@/components/UserStats.vue'
import { api } from '@/lib/api'
import type { User } from '@/types'
import { useUserResources } from '@/composables/userResources'

const { fetchUserResources, userResourcesGB } = useUserResources(api)

const whoami = ref<User | null>(null)
const stats = computed(() => {
  if (!userResourcesGB.value) {
    return []
  }

  return [
    {
      item: 'CPU',
      active: userResourcesGB.value.active_vms_cores,
      max: userResourcesGB.value.max_cores,
      allocated: userResourcesGB.value.allocated_cores,
      group_max: userResourcesGB.value.group_max_cores,
    },
    {
      item: 'RAM',
      active: userResourcesGB.value.active_vms_ram,
      max: userResourcesGB.value.max_ram,
      allocated: userResourcesGB.value.allocated_ram,
      group_max: userResourcesGB.value.group_max_ram,
    },
    {
      item: 'Disk',
      active: userResourcesGB.value.active_vms_disk,
      max: userResourcesGB.value.max_disk,
      allocated: userResourcesGB.value.allocated_disk,
      group_max: userResourcesGB.value.group_max_disk,
    },
    {
      item: 'Net',
      active: -1,
      max: userResourcesGB.value.max_nets,
      allocated: userResourcesGB.value.allocated_nets,
      group_max: userResourcesGB.value.group_max_nets,
    },
  ]
})

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      whoami.value = res.data as User
    })
    .catch((err) => {
      if (err.response && err.response.status === 401) {
        // Could be first load and user is not logged in, ignore the error
        return
      }
      console.error('Failed to fetch whoami:', err)
    })
}

onMounted(() => {
  fetchWhoami()
  fetchUserResources()
})
</script>

<template>
  <div class="h-full overflow-auto p-2">
    <div class="mb-6">
      <div class="flex justify-between">
        <h1 class="flex items-center gap-3 text-3xl font-bold">
          <IconVue class="text-primary" icon="material-symbols:home-rounded"></IconVue>
          Hi {{ whoami?.username }}!
        </h1>
        <HelpButton />
      </div>
      <h2 class="text-base-content/80 my-2 text-xl font-semibold">Usage of your resources</h2>
    </div>

    <UserStats v-if="stats.length !== 0" :stats="stats" />

    <div v-else class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>
  </div>
</template>
