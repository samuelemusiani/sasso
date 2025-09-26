<script lang="ts" setup>
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { Backup } from '@/types'
import { api } from '@/lib/api'

const backups = ref<Backup[]>([])

const route = useRoute()
const vmid = Number(route.params.vmid)

function fetchBackups() {
  api
    .get(`/vm/${vmid}/backup`)
    .then((res) => {
      // Handle the response data
      backups.value = res.data as Backup[]
      console.log('Fetched backups:', backups)
    })
    .catch((err) => {
      console.error('Failed to fetch backups:', err)
    })
}

function restoreBackup(backupName: string) {
  if (
    confirm(
      `Are you sure you want to restore backup ${backupName}? This will overwrite the current VM state.`,
    )
  ) {
    api
      .post(`/vm/${vmid}/backup/${backupName}/restore`)
      .then(() => {
        console.log('Backup restoring')
      })
      .catch((err) => {
        console.error('Failed to restore backup:', err)
        alert(`Failed to restore backup ${backupName}.`)
      })
  }
}

function deleteBackup(backupName: string) {
  if (
    confirm(`Are you sure you want to delete backup ${backupName}? This action cannot be undone.`)
  ) {
    api
      .delete(`/vm/${vmid}/backup/${backupName}`)
      .then(() => {
        console.log('Backup deleted')
        fetchBackups() // Refresh the list after deletion
      })
      .catch((err) => {
        console.error('Failed to delete backup:', err)
        alert(`Failed to delete backup ${backupName}.`)
      })
  }
}

function makeBackup() {
  api
    .post(`/vm/${vmid}/backup`)
    .then(() => {
      console.log('Backup created')
    })
    .catch((err) => {
      console.error('Failed to create backup:', err)
      alert(`Failed to create backup`)
    })
}

onMounted(() => {
  fetchBackups()
})
</script>

<template>
  <div>This is the Backups view for <b>sasso</b>!</div>
  <RouterLink
    :to="`/vm/`"
    class="bg-blue-500 p-2 rounded-lg hover:bg-blue-400 text-white mb-4 inline-block"
  >
    Back to VMs
  </RouterLink>
  <button
    @click="makeBackup()"
    class="bg-green-500 p-2 rounded-lg hover:bg-green-400 text-white mb-4 inline-block"
  >
    Create Backup
  </button>
  <div class="overflow-x-auto">
    <table class="min-w-full divide-y divide-gray-200">
      <thead class="bg-gray-50">
        <tr>
          <th
            scope="col"
            class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
          >
            ID
          </th>
          <th
            scope="col"
            class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
          >
            Time
          </th>
          <th scope="col" class="relative px-6 py-3"><span class="sr-only">Actions</span></th>
        </tr>
      </thead>
      <tbody class="bg-white divide-y divide-gray-200">
        <tr v-for="bk in backups" :key="bk.name">
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.name.substring(0, 10) }}</td>
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.ctime }}</td>
          <td
            class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium flex gap-2 justify-end"
          >
            <button
              @click="restoreBackup(bk.name)"
              class="bg-yellow-400 p-2 rounded-lg hover:bg-yellow-300 text-white"
            >
              Restore
            </button>
            <button
              v-if="bk.can_delete"
              @click="deleteBackup(bk.name)"
              class="bg-red-400 p-2 rounded-lg hover:bg-red-300 text-white"
            >
              Delete
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
