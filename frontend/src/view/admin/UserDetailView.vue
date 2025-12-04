<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/lib/api'
import type { User } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import { useToastService } from '@/composables/useToast'

const { error: toastError, success: toastSuccess } = useToastService()

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
      maxCores.value = user.value.max_cores ?? 0
      maxRAM.value = user.value.max_ram ?? 0
      maxDisk.value = user.value.max_disk ?? 0
      maxNets.value = user.value.max_nets ?? 0
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
      toastSuccess('User limits updated successfully')
      router.push('/admin/users') // Redirect back to the user list
    })
    .catch((err) => {
      console.error('Failed to update user limits:', err)
      toastError('Failed to update user limits')
    })
}

onMounted(() => {
  fetchUser()
})
</script>

<template>
  <div class="p-2">
    <AdminBreadcrumbs />
    <h2 class="text-2xl font-bold">User Details</h2>

    <div v-if="user" class="mt-4">
      <p><strong>ID</strong> {{ user.id }}</p>
      <p><strong>Username</strong> {{ user.username }}</p>
      <p><strong>Email</strong> {{ user.email }}</p>
      <p><strong>Realm</strong> {{ user.realm }}</p>
      <p><strong>Role</strong> {{ user.role }}</p>

      <h3 class="mt-6 text-xl font-bold">Resource Limits</h3>
      <form @submit.prevent="updateLimits" class="mt-4 space-y-4">
        <div>
          <label for="maxCores" class="block text-sm font-medium">Max Cores</label>
          <input type="number" id="maxCores" v-model.number="maxCores" class="input rounded-lg" />
        </div>
        <div>
          <label for="maxRAM" class="block text-sm font-medium">Max RAM (MB)</label>
          <input type="number" id="maxRAM" v-model.number="maxRAM" class="input rounded-lg" />
        </div>
        <div>
          <label for="maxDisk" class="block text-sm font-medium">Max Disk (GB)</label>
          <input type="number" id="maxDisk" v-model.number="maxDisk" class="input rounded-lg" />
        </div>
        <div>
          <label for="maxNets" class="block text-sm font-medium">Max Networks</label>
          <input type="number" id="maxNets" v-model.number="maxNets" class="input rounded-lg" />
        </div>
        <button type="submit" class="btn btn-primary rounded-lg">Update Limits</button>
      </form>
    </div>
    <div v-else>
      <p>Loading user details...</p>
    </div>
  </div>
</template>
