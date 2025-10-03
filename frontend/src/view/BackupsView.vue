<script lang="ts" setup>
import { onMounted, ref, computed, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import type { Backup, BackupRequest } from '@/types'
import { api } from '@/lib/api'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import CreateNew from '@/components/CreateNew.vue'

const backups = ref<Backup[]>([])

const name = ref('')
const notes = ref('')

const route = useRoute()
const vmid = Number(route.params.vmid)
const error = ref('')

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
    // FIXME: pls remove this stupid id
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
      error.value = 'Failed to create backup: ' + err.message
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
  <!-- TODO: go back to vm -->
  <div class="flex gap-2 flex-col">
    <h2 class="text-2xl font-bold">Create a Backup</h2>
    <CreateNew :create="makeBackup" title="New Backup" :error="error">
      <label class="label">Backup Name</label>
      <input type="text" placeholder="Name" v-model="name" class="input rounded-lg w-full" />
      <label class="label">Backup Notes</label>
      <textarea placeholder="Notes" v-model="notes" class="input rounded-lg w-full h-32"></textarea>
    </CreateNew>
    <div v-show="pendingBackupRequests.length > 0">
      <!-- TODO: quando ne crei uno resta in pending per un po' e non si vede subito nella tabella -->
      <!-- TODO: dopo il pending rifaccio refetch -->
      {{ pendingBackupRequests[0] }}
    </div>
    <div class="overflow-x-auto">
      <table class="table min-w-full divide-y">
        <thead>
          <tr>
            <th scope="col" class="font-medium uppercase">
              Name
            </th>
            <th scope="col" class="font-medium uppercase">
              Time
            </th>
            <th scope="col" class="font-medium uppercase">
              Notes
            </th>
            <th scope="col" class="font-medium uppercase">
              Protected
            </th>
            <th scope="col" class="relative px-6 py-3"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="bk in backups" :key="bk.name">
            <td>{{ bk.name }}</td>
            <!-- TODO: format timing -->
            <td>{{ bk.ctime }}</td>
            <!-- TODO: fix with some fancy notes -->
            <td>{{ bk.notes }}</td>
            <td>{{ bk.protected }}</td>
            <td class="text-right text-sm font-medium flex gap-2 justify-end">
              <!-- TODO: add info: evita che un backup venga eliminato da un jb di pruning automatico -->
              <button @click="protectBackup(bk.id, !bk.protected)" class="btn btn-primary rounded-lg">
                {{ bk.protected ? 'Unprotect' : 'Protect' }}
              </button>
              <button @click="restoreBackup(bk.id)" class="btn btn-warning rounded-lg">
                Restore
              </button>
              <button v-if="bk.can_delete" @click="deleteBackup(bk.id)" class="btn btn-error btn-outline rounded-lg">
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
