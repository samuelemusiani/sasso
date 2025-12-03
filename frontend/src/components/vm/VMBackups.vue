<script lang="ts" setup>
import { onMounted, ref, computed, onBeforeUnmount, watch } from 'vue'
import { useRoute } from 'vue-router'
import type { Backup, BackupRequest, VM } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { useLoadingStore } from '@/stores/loading'
import { getStatusClass } from '@/const'
import { formatDate } from '@/lib/utils'
import { useToastService } from '@/composables/useToast'

const { error: toastError, success: toastSuccess } = useToastService()

const $props = defineProps<{
  vm: VM
}>()

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

const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

function fetchBackupsRequests() {
  return api
    .get(`/vm/${vmid}/backup/request`)
    .then((res) => {
      // Handle the response data
      backupRequests.value = res.data as BackupRequest[]
    })
    .catch((err) => {
      console.error('Failed to fetch backup requests:', err)
    })
}

function fetchBackups() {
  loading.start('vm', vmid, 'fetch_backups')
  api
    .get(`/vm/${vmid}/backup`)
    .then((res) => {
      // Handle the response data
      const tmp = res.data.sort((a: Backup, b: Backup) => {
        return new Date(b.ctime).getTime() - new Date(a.ctime).getTime()
      })
      backups.value = tmp as Backup[]
    })
    .catch((err) => {
      console.error('Failed to fetch backups:', err)
    })
    .finally(() => {
      loading.stop('vm', vmid, 'fetch_backups')
    })
}

function restoreBackup(backupID: string) {
  if (
    confirm(
      `Are you sure you want to restore this backup? This will overwrite the current VM state.`,
    )
  ) {
    api
      .post(`/vm/${vmid}/backup/${backupID}/restore`)
      .then(() => {
        fetchBackupsRequests()
      })
      .catch((err) => {
        console.error('Failed to restore backup:', err)
      })
  }
}

function deleteBackup(backupID: string) {
  loading.start('backup', backupID, 'delete')
  if (confirm(`Are you sure you want to delete this backup? This action cannot be undone.`)) {
    api
      .delete(`/vm/${vmid}/backup/${backupID}`)
      .then(() => {
        toastSuccess(`Backup deletion request submitted.`)
        fetchBackupsRequests()
      })
      .catch((err) => {
        console.error('Failed to delete backup:', err)
        toastError(`Failed to send delete request for backup.`)
      })
      .finally(() => {
        loading.stop('backup', backupID, 'delete')
      })
  }
}

function protectBackup(backupID: string, protect: boolean) {
  loading.start('backup', backupID, 'protect')
  api
    .post(`/vm/${vmid}/backup/${backupID}/protect`, {
      protected: protect,
    })
    .then(() => {
      console.log('Backup protection toggled')
      backups.value = backups.value.map((bk) =>
        bk.id === backupID ? { ...bk, protected: protect } : bk,
      )
      fetchBackups() // Refresh the list after deletion
      toastSuccess(`Backup ${backupID} is now ${protect ? 'protected' : 'unprotected'}.`)
    })
    .catch((err) => {
      console.error('Failed to toggle backup protection:', err)
      toastError(`Failed to toggle protection for backup.`)
    })
    .finally(() => {
      loading.stop('backup', backupID, 'protect')
    })
}

function makeBackup() {
  loading.start('vm', vmid, 'create_backup')
  api
    .post(`/vm/${vmid}/backup`, {
      name: name.value,
      notes: notes.value,
    })
    .then(() => {
      console.log('Backup created')
      fetchBackupsRequests()
      toastSuccess('Backup creation request submitted.')
    })
    .catch((err) => {
      error.value = 'Failed to create backup: ' + err.response.data
      console.error('Failed to create backup:', err)
      toastError('Failed to send backup creation request.')
    })
    .finally(() => {
      loading.stop('vm', vmid, 'create_backup')
    })
}

const backupMessage = computed(() => {
  if (pendingBackupRequests.value.length > 0) {
    const req = pendingBackupRequests.value[0]
    if (!req) return ''

    if (req.type === 'create') {
      return 'A backup is being created. The page will refresh automatically when it is done. Please wait...'
    } else if (req.type === 'restore') {
      return 'A backup is being restored. The page will refresh automatically when it is done. Please wait...'
    } else if (req.type === 'delete') {
      return 'A backup is being deleted. The page will refresh automatically when it is done. Please wait...'
    }
  }
  return ''
})

watch(pendingBackupRequests, (newVal, oldVal) => {
  if (oldVal.length > 0 && newVal.length === 0) {
    // All pending requests are done
    fetchBackups()
  }
})

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
  <div class="flex flex-col gap-2">
    <CreateNew
      v-if="$props.vm.group_role !== 'member'"
      :create="makeBackup"
      title="New Backup"
      :error="error"
      :loading="isLoading(vm.id, 'create_backup')"
    >
      <label class="label">Backup Name</label>
      <input type="text" placeholder="Name" v-model="name" class="input w-full rounded-lg" />
      <label class="label">Backup Notes</label>
      <textarea placeholder="Notes" v-model="notes" class="input h-32 w-full rounded-lg"></textarea>
    </CreateNew>
    <div>
      {{ backupMessage }}
    </div>
    <div v-if="isLoading(vm.id, 'fetch_backups')" class="grid h-70">
      <span class="loading loading-spinner place-self-center"></span>
    </div>
    <div v-else class="overflow-x-auto">
      <table class="table min-w-full divide-y">
        <thead>
          <tr>
            <th scope="col" class="font-medium uppercase">Name</th>
            <th scope="col" class="font-medium uppercase">Time</th>
            <th scope="col" class="font-medium uppercase">Notes</th>
            <th scope="col" class="font-medium uppercase">Protected</th>
            <th scope="col" class="relative px-6 py-3"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="bk in backups" :key="bk.name">
            <td>{{ bk.name }}</td>
            <td>{{ formatDate(bk.ctime) }}</td>
            <!-- TODO: fix with some fancy notes -->
            <td>{{ bk.notes }}</td>
            <td class="font-semibold capitalize" :class="getStatusClass(bk.protected.toString())">
              {{ bk.protected }}
            </td>
            <td
              v-if="$props.vm.group_role !== 'member'"
              class="flex justify-end gap-2 text-right text-sm font-medium"
            >
              <!-- TODO: add info: evita che un backup venga eliminato da un jb di pruning automatico -->
              <button
                @click="protectBackup(bk.id, !bk.protected)"
                class="btn btn-primary rounded-lg"
                :disabled="loading.is('backup', bk.id, 'protect')"
              >
                <span
                  v-show="loading.is('backup', bk.id, 'protect')"
                  class="loading loading-spinner loading-xs"
                ></span>
                {{ bk.protected ? 'Unprotect' : 'Protect' }}
              </button>
              <button @click="restoreBackup(bk.id)" class="btn btn-warning rounded-lg">
                Restore
              </button>
              <button
                v-if="bk.can_delete"
                @click="deleteBackup(bk.id)"
                class="btn btn-error btn-outline rounded-lg"
                :disabled="loading.is('backup', bk.id, 'delete')"
              >
                <span
                  v-show="loading.is('backup', bk.id, 'delete')"
                  class="loading loading-spinner loading-xs"
                ></span>
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
