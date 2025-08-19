<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import type { Whoami } from '@/types'
import { api } from '@/lib/api'

const whoami = ref<Whoami | null>(null)

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      console.log('Whoami response:', res.data)
      whoami.value = res.data as Whoami
    })
    .catch((err) => {
      console.error('Failed to fetch whoami:', err)
    })
}

const showAdminPanel = computed(() => {
  return whoami.value?.role == 'admin' ?? false
})

onMounted(() => {
  fetchWhoami()
})
</script>

<template>
  <div class="p-2">
    <div>Home view for <b>sasso</b>!</div>
    <div v-if="whoami">
      {{ whoami }}
    </div>
    <div class="flex gap-2">
      <RouterLink
        class="bg-green-400 hover:bg-green-300 p-2 rounded-lg min-w-32 block text-center"
        to="/vm"
      >
        VM
      </RouterLink>
      <RouterLink
        class="bg-blue-400 hover:bg-blue-300 p-2 rounded-lg min-w-32 block text-center"
        to="/login"
      >
        Login
      </RouterLink>
      <RouterLink
        v-if="showAdminPanel"
        class="bg-gray-400 hover:bg-gray-300 p-2 rounded-lg min-w-32 block text-center"
        to="/admin"
      >
        Admin pannel
      </RouterLink>
    </div>
  </div>
</template>
