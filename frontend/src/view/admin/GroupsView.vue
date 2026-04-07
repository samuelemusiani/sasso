<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Group } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const groups = ref<Group[]>([])

function fetchGroups() {
  loading.start('groups', null, 'fetch')
  api
    .get('/admin/groups')
    .then((res) => {
      console.log(res.data)
      groups.value = res.data as Group[]
    })
    .catch((err) => {
      console.error('Failed to fetch groups:', err)
      toastError('Failed to fetch groups: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('groups', null, 'fetch')
    })
}

onMounted(() => {
  fetchGroups()
})
</script>

<template>
  <div class="p-2">
    <div class="flex justify-between">
      <AdminBreadcrumbs />
      <HelpButton />
    </div>

    <div v-if="loading.is('groups', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="mt-2 table w-full p-2">
      <thead>
        <tr class="uppercase">
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
            <RouterLink
              :to="`/admin/groups/${group.id}`"
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
