<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { User } from '@/types'
import BreadcrumbNav from '@/components/BreadcrumbNav.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const users = ref<User[]>([])

function fetchUsers() {
  loading.start('users', null, 'fetch')
  api
    .get('/admin/users')
    .then((res) => {
      users.value = res.data.sort((a: User, b: User) => a.id - b.id) as User[]
    })
    .catch((err) => {
      console.error('Failed to fetch users:', err)
      toastError('Failed to fetch users: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('users', null, 'fetch')
    })
}

onMounted(() => {
  fetchUsers()
})
</script>

<template>
  <div class="p-2">
    <div class="flex justify-between px-4">
      <BreadcrumbNav />
      <HelpButton />
    </div>

    <div v-if="loading.is('users', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="mt-2 table w-full p-2">
      <thead>
        <tr class="uppercase">
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
            <RouterLink
              :to="`/admin/users/${user.id}`"
              class="btn btn-primary btn-sm md:btn-md rounded-lg"
            >
              <IconVue icon="material-symbols:edit" class="text-lg" />
              <p class="hidden md:inline">Edit</p>
            </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
