<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { User } from '@/types'

const users = ref<User[]>([])

function fetchUsers() {
  api
    .get('/admin/users')
    .then((res) => {
      users.value = res.data as User[]
    })
    .catch((err) => {
      console.error('Failed to fetch users:', err)
    })
}

onMounted(() => {
  fetchUsers()
})
</script>

<template>
  <div class="p-2">
    <div>Admin users view for <b>sasso</b>!</div>
    <RouterLink
      class="bg-gray-400 hover:bg-gray-300 p-2 rounded-lg w-64 block text-center"
      to="/admin"
    >
      Back to Admin Panel
    </RouterLink>
    <table class="w-full mt-2 p-2">
      <thead>
        <tr class="bg-cyan-500">
          <th class="p-2 border-y border-black border-l">ID</th>
          <th class="p-2 border-y border-black">Username</th>
          <th class="p-2 border-y border-black">Email</th>
          <th class="p-2 border-y border-black">Role</th>
          <th class="p-2 border-y border-black border-r">Realm</th>
          <th class="p-2 border-y border-black border-r">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id" class="odd:bg-blue-100 even:bg-blue-200">
          <td class="p-2 text-center border-y border-black border-l">{{ user.id }}</td>
          <td class="p-2 text-center border-y border-black">{{ user.username }}</td>
          <td class="p-2 text-center border-y border-black">{{ user.email }}</td>
          <td class="p-2 text-center border-y border-black">{{ user.role }}</td>
          <td class="p-2 text-center border-y border-black">{{ user.realm }}</td>
          <td class="p-2 text-center border-y border-black border-r">
            <RouterLink :to="`/admin/users/${user.id}`" class="text-blue-600 hover:underline">
              Edit
            </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
