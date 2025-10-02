<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { User } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'

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
    <AdminBreadcrumbs />
    <table class="table w-full mt-2 p-2">
      <thead>
        <tr class="">
          <th class="">Username</th>
          <th class="">Email</th>
          <th class="">Role</th>
          <th class="">Realm</th>
          <th class="">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.id" class="odd:bg-base-100 even:bg-base-200">
          <td class="">{{ user.username }}</td>
          <td class="">{{ user.email }}</td>
          <td class="">{{ user.role }}</td>
          <td class="">{{ user.realm }}</td>
          <td class="">
            <RouterLink :to="`/admin/users/${user.id}`" class="btn btn-primary"> Edit </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
