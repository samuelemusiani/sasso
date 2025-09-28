<script lang="ts" setup>
import { onMounted, ref, computed, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import type { Backup, BackupRequest } from '@/types'
import { api } from '@/lib/api'

const backups = ref<Backup[]>([])

const name = ref('')
const notes = ref('')

const route = useRoute()
const vmid = Number(route.params.vmid)

const backupRequests = ref<BackupRequest[]>([])
const pendingBackupRequests = computed(() =>
  backupRequests.value.filter((req) => req.status === 'pending'),
)

function fetchBackupsRequests() {
  return api
    .get(`/vm/${vmid}/backup/request`)
    .then((res) => {
      // Handle the response data
      backupRequests.value = res.data as BackupRequest[]
      console.log('Fetched backup requests:', backupRequests)
    })
    .catch((err) => {
      console.error('Failed to fetch backup requests:', err)
    })
}

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

function restoreBackup(backupID: string) {
  if (
    confirm(
      `Are you sure you want to restore backup ${backupID}? This will overwrite the current VM state.`,
    )
  ) {
    api
      .post(`/vm/${vmid}/backup/${backupID}/restore`)
      .then(() => {
        console.log('Backup restoring')
        fetchBackupsRequests()
      })
      .catch((err) => {
        console.error('Failed to restore backup:', err)
        alert(`Failed to restore backup ${backupID}.`)
      })
  }
}

function deleteBackup(backupID: string) {
  if (
    confirm(`Are you sure you want to delete backup ${backupID}? This action cannot be undone.`)
  ) {
    api
      .delete(`/vm/${vmid}/backup/${backupID}`)
      .then(() => {
        console.log('Backup deleted')
        fetchBackups() // Refresh the list after deletion
        fetchBackupsRequests()
      })
      .catch((err) => {
        console.error('Failed to delete backup:', err)
        alert(`Failed to delete backup ${backupID}.`)
      })
  }
}

function protectBackup(backupID: string, protect: boolean) {
  api
    .post(`/vm/${vmid}/backup/${backupID}/protect`, {
      protected: protect,
    })
    .then(() => {
      console.log('Backup protection toggled')
      fetchBackups() // Refresh the list after deletion
    })
    .catch((err) => {
      console.error('Failed to toggle backup protection:', err)
      alert(`Failed to toggle backup protection for ${backupID}.`)
    })
}

function makeBackup() {
  api
    .post(`/vm/${vmid}/backup`, {
      name: name.value,
      notes: notes.value,
    })
    .then(() => {
      console.log('Backup created')
      fetchBackupsRequests()
    })
    .catch((err) => {
      console.error('Failed to create backup:', err)
      alert(`Failed to create backup`)
    })
}

let intervalId: number | null = null

onMounted(() => {
  fetchBackups()
  fetchBackupsRequests()
  intervalId = setInterval(() => {
    fetchBackupsRequests()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
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
  <input type="text" placeholder="Backup Name" v-model="name" class="border p-2 rounded-lg mb-4" />
  <input
    type="text"
    placeholder="Backup Notes"
    v-model="notes"
    class="border p-2 rounded-lg mb-4"
  />
  <button
    @click="makeBackup()"
    class="bg-green-500 p-2 rounded-lg hover:bg-green-400 text-white mb-4 inline-block"
  >
    Create Backup
  </button>
  <div>
    {{ pendingBackupRequests }}
  </div>
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
            Name
          </th>
          <th
            scope="col"
            class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
          >
            Time
          </th>
          <th
            scope="col"
            class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
          >
            Notes
          </th>
          <th
            scope="col"
            class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
          >
            Protected
          </th>
          <th scope="col" class="relative px-6 py-3"><span class="sr-only">Actions</span></th>
        </tr>
      </thead>
      <tbody class="bg-white divide-y divide-gray-200">
        <tr v-for="bk in backups" :key="bk.name">
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.id.substring(0, 10) }}</td>
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.name }}</td>
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.ctime }}</td>
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.notes }}</td>
          <td class="px-6 py-4 whitespace-nowrap">{{ bk.protected }}</td>
          <td
            class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium flex gap-2 justify-end"
          >
            <button
              @click="protectBackup(bk.id, !bk.protected)"
              class="bg-blue-400 p-2 rounded-lg hover:bg-blue-300 text-white"
            >
              Protect
            </button>
            <button
              @click="restoreBackup(bk.id)"
              class="bg-yellow-400 p-2 rounded-lg hover:bg-yellow-300 text-white"
            >
              Restore
            </button>
            <button
              v-if="bk.can_delete"
              @click="deleteBackup(bk.id)"
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
