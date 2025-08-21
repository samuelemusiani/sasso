<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import type { User } from '@/types'
import { api } from '@/lib/api'
import { useRouter } from 'vue-router'

const whoami = ref<User | null>(null)
const router = useRouter()

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      console.log('Whoami response:', res.data)
      whoami.value = res.data as User
    })
    .catch((err) => {
      console.error('Failed to fetch whoami:', err)
    })
}

const showAdminPanel = computed(() => {
  if (!whoami.value) return false
  return whoami.value.role === 'admin'
})

function logout() {
  localStorage.removeItem('jwt_token')
  router.push('/login')
}

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
      <button
        @click="logout"
        class="bg-red-400 hover:bg-red-300 p-2 rounded-lg min-w-32 block text-center"
      >
        Logout
      </button>
    </div>
  </div>
</template>
