<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/lib/api'
import type { User } from '@/types'
import { globalNotifications } from '@/lib/notifications'

const route = useRoute()
const router = useRouter()
const user = ref<User | null>(null)
const maxCores = ref<number>(0)
const maxRAM = ref<number>(0)
const maxDisk = ref<number>(0)
const maxNets = ref<number>(0)

function fetchUser() {
  const userId = route.params.id
  api
    .get(`/admin/users/${userId}`)
    .then((res) => {
      user.value = res.data as User
      maxCores.value = user.value.max_cores
      maxRAM.value = user.value.max_ram
      maxDisk.value = user.value.max_disk
      maxNets.value = user.value.max_nets
    })
    .catch((err) => {
      console.error('Failed to fetch user:', err)
      // Optionally redirect to a 404 page or show an error message
    })
}

function updateLimits() {
  if (!user.value) return

  api
    .put('/admin/users/limits', {
      user_id: user.value.id,
      max_cores: maxCores.value,
      max_ram: maxRAM.value,
      max_disk: maxDisk.value,
      max_nets: maxNets.value,
    })
    .then(() => {
      globalNotifications.showSuccess('User limits updated successfully!')
      router.push('/admin/users') // Redirect back to the user list
    })
    .catch((err) => {
      console.error('Failed to update user limits:', err)
      globalNotifications.showError('Failed to update user limits.')
    })
}

onMounted(() => {
  fetchUser()
})
</script>

<template>
  <div class="p-2">
    <RouterLink
      class="bg-gray-400 hover:bg-gray-300 p-2 rounded-lg w-64 block text-center"
      to="/admin/users"
    >
      Back to User List
    </RouterLink>

    <h2 class="text-2xl font-bold mt-4">User Details</h2>

    <div v-if="user" class="mt-4">
      <p><strong>ID:</strong> {{ user.id }}</p>
      <p><strong>Username:</strong> {{ user.username }}</p>
      <p><strong>Email:</strong> {{ user.email }}</p>
      <p><strong>Realm:</strong> {{ user.realm }}</p>
      <p><strong>Role:</strong> {{ user.role }}</p>

      <h3 class="text-xl font-bold mt-6">Resource Limits</h3>
      <form @submit.prevent="updateLimits" class="mt-4 space-y-4">
        <div>
          <label for="maxCores" class="block text-sm font-medium text-gray-700">Max Cores:</label>
          <input
            type="number"
            id="maxCores"
            v-model.number="maxCores"
            class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
          />
        </div>
        <div>
          <label for="maxRAM" class="block text-sm font-medium text-gray-700">Max RAM (MB):</label>
          <input
            type="number"
            id="maxRAM"
            v-model.number="maxRAM"
            class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
          />
        </div>
        <div>
          <label for="maxDisk" class="block text-sm font-medium text-gray-700"
            >Max Disk (MB):</label
          >
          <input
            type="number"
            id="maxDisk"
            v-model.number="maxDisk"
            class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
          />
        </div>
        <div>
          <label for="maxNets" class="block text-sm font-medium text-gray-700">Max Networks:</label>
          <input
            type="number"
            id="maxNets"
            v-model.number="maxNets"
            class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
          />
        </div>
        <button
          type="submit"
          class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
        >
          Update Limits
        </button>
      </form>
    </div>
    <div v-else>
      <p>Loading user details...</p>
    </div>
  </div>
</template>
