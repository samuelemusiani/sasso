<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Group } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'

const groups = ref<Group[]>([])

function fetchGroups() {
  api
    .get('/admin/groups')
    .then((res) => {
      console.log(res.data)
      groups.value = res.data as Group[]
    })
    .catch((err) => {
      console.error('Failed to fetch groups:', err)
    })
}

onMounted(() => {
  fetchGroups()
})
</script>

<template>
  <div class="p-2">
    <AdminBreadcrumbs />
    <table class="mt-2 table w-full p-2">
      <thead>
        <tr class="">
          <th class="">ID</th>
          <th class="">Name</th>
          <th class="">Description</th>
          <th class="">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="group in groups" :key="group.id" class="odd:bg-base-100 even:bg-base-200">
          <td class="">{{ group.id }}</td>
          <td class="">{{ group.name }}</td>
          <td class="">{{ group.description }}</td>
          <td class="">
            <RouterLink :to="`/admin/groups/${group.id}`" class="btn btn-primary">
              Edit
            </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
