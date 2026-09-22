<script setup lang="ts">
import { onMounted, ref, onBeforeUnmount } from 'vue'
import { api } from '@/lib/api'
import { useRoute } from 'vue-router'
import { formatDate } from '@/lib/utils'
import { useLoadingStore } from '@/stores/loading'
import type { BackupRequest } from '@/types'

const route = useRoute()
const vmid = Number(route.params.vmid)
const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

const backupRequests = ref<BackupRequest[]>([])

function fetchBackupsRequests() {
  loading.start('vm', vmid, 'fetch-backup-requests')
  return api
    .get(`/vm/${vmid}/backup/request?vmid=${vmid}`)
    .then((res) => {
      // Handle the response data
      const tmp = res.data as BackupRequest[]
      backupRequests.value = tmp.sort((a, b) => b.id - a.id)
    })
    .catch((err) => {
      console.error('Failed to fetch backup requests:', err)
    })
    .finally(() => {
      loading.stop('vm', vmid, 'fetch-backup-requests')
    })
}

function fetchBackupRequestWithLoading() {
  return api
    .get(`/vm/${vmid}/backup/request?vmid=${vmid}`)
    .then((res) => {
      // Handle the response data
      const tmp = res.data as BackupRequest[]
      backupRequests.value = tmp.sort((a, b) => b.id - a.id)
    })
    .catch((err) => {
      console.error('Failed to fetch backup requests:', err)
    })
}

const truncateLength = 50

function truncateString(s: string, length: number = truncateLength) {
  if (s.length > length) {
    return s.substring(0, length - 3) + '...'
  }
  return s
}

function statusClass(status: string) {
  switch (status) {
    case 'pending':
      return 'text-warning'
    case 'completed':
      return 'text-success'
    case 'failed':
      return 'text-error'
    default:
      return ''
  }
}

function statusIcon(status: string) {
  switch (status) {
    case 'pending':
      return 'material-symbols:hourglass-top'
    case 'completed':
      return 'material-symbols:check-circle'
    case 'failed':
      return 'material-symbols:error'
    default:
      return 'material-symbols:help'
  }
}

function typeClass(type: string) {
  switch (type) {
    case 'create':
      return 'text-success'
    case 'restore':
      return 'text-warning'
    case 'delete':
      return 'text-error'
    default:
      return ''
  }
}

function typeIcon(type: string) {
  switch (type) {
    case 'create':
      return 'material-symbols:add-circle'
    case 'restore':
      return 'material-symbols:settings-backup-restore'
    case 'delete':
      return 'material-symbols:delete'
    default:
      return 'material-symbols:help'
  }
}

const dialogRef = ref<HTMLDialogElement | null>(null)
function closeModal() {
  const el = dialogRef.value
  if (!el) return
  if (el.open) el.close()
}

function openModal() {
  const el = dialogRef.value
  if (!el) return
  if (!el.open) el.showModal()
}

const backupIdToShow = ref<string>('')

function showFullBackupId(backupId: string) {
  backupIdToShow.value = backupId
  openModal()
}

let intervalId: number | null = null

onMounted(() => {
  fetchBackupsRequests()
  intervalId = setInterval(() => {
    fetchBackupRequestWithLoading()
  }, 2000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="text-xl font-bold capitalize">Backup requests history</div>
    <div v-if="isLoading(vmid, 'fetch-backup-requests')" class="grid h-70">
      <span class="loading loading-spinner text-primary place-self-center"></span>
    </div>
    <table v-else class="table min-w-full divide-y">
      <thead>
        <tr>
          <th scope="col" class="font-medium uppercase">id</th>
          <th scope="col" class="font-medium uppercase">created at</th>
          <th scope="col" class="font-medium uppercase">backup id</th>
          <th scope="col" class="font-medium uppercase">type</th>
          <th scope="col" class="font-medium uppercase">status</th>
          <!--th scope="col" class="font-medium uppercase">vmid</th-->
          <th scope="col" class="font-medium uppercase">name</th>
          <!--th scope="col" class="font-medium uppercase">notes</th-->
        </tr>
      </thead>
      <tbody class="divide-y">
        <tr v-for="br in backupRequests" :key="br.id" class="">
          <td>{{ br.id }}</td>
          <td>{{ formatDate(br.created_at) }}</td>
          <td>
            <div class="hover:link" @click="showFullBackupId(br.backup_id)">
              {{ truncateString(br.backup_id, 16) }}
            </div>
          </td>
          <td>
            <div class="flex items-center gap-2">
              <IconVue :icon="typeIcon(br.type)" :class="[typeClass(br.type)]" class="text-lg" />
              <div :class="[typeClass(br.type)]" class="font-medium capitalize">
                {{ br.type }}
              </div>
            </div>
          </td>
          <td>
            <div class="flex items-center gap-2">
              <IconVue
                :icon="statusIcon(br.status)"
                :class="[statusClass(br.status)]"
                class="text-lg"
              />
              <div :class="[statusClass(br.status)]" class="font-medium capitalize">
                {{ br.status }}
              </div>
            </div>
          </td>
          <td>{{ br.name }}</td>
          <!-- td>{{ truncateString(br.notes ?? '', 20) }}</td-->
        </tr>
      </tbody>
    </table>

    <dialog ref="dialogRef" class="modal modal-bottom sm:modal-middle">
      <div class="modal-box">
        <h3 class="mb-4 text-xl font-bold">Full Backup ID</h3>
        <p class="font-mono break-all">{{ backupIdToShow }}</p>

        <div class="modal-action">
          <button class="btn" type="button" @click="closeModal()">Close</button>
        </div>
      </div>

      <!-- Optional backdrop click closes the dialog (counts as negative via close-sync + cancel/close behavior) -->
      <form method="dialog" class="modal-backdrop">
        <button aria-label="Close"></button>
      </form>
    </dialog>
  </div>
</template>
