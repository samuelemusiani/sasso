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
import ModalAlert from '@/components/ModalAlert.vue'

const { error: toastError, success: toastSuccess } = useToastService()

const $props = defineProps<{
  vm: VM
}>()

const backups = ref<Backup[]>([])

const name = ref('')
const notes = ref('')

const notesModalRef = ref<HTMLDialogElement | null>(null)
const notesModalTitle = ref('')
const notesModalBody = ref('')

const route = useRoute()
const vmid = Number(route.params.vmid)
const error = ref('')

// This values is hardcoded in the backend
const maxProtectedBackupsForUser = 4

function numberOfProtectedBackups() {
  return backups.value.filter((bk) => bk.protected).length
}

function haveFinishedProtectedBackups() {
  const protectedBackups = numberOfProtectedBackups()
  return protectedBackups >= maxProtectedBackupsForUser
}

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

function fetchBackupsWithoutLoading() {
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
}

const showRestoreModal = ref(false)
const backupToRestore = ref<string | null>(null)

function restoreBackup(backupID: string) {
  api
    .post(`/vm/${vmid}/backup/${backupID}/restore`)
    .then(() => {
      fetchBackupsRequests()
    })
    .catch((err) => {
      console.error('Failed to restore backup:', err)
      toastError(`Failed to send restore request for backup. ${err.response.data}`)
    })
    .finally(() => {
      loading.stop('backup', backupID, 'restore')
    })
}

function cancelRestoreBackup(backupID: string) {
  showRestoreModal.value = false
  loading.stop('backup', backupID, 'restore')
}

function preRestoreBackup(backupID: string) {
  backupToRestore.value = backupID
  showRestoreModal.value = true
  loading.start('backup', backupID, 'restore')

  // Wait for the user to confirm restoration in the modal
}

const showDeleteModal = ref(false)
const backupToDelete = ref<string | null>(null)

function preDeleteBackup(backupID: string) {
  backupToDelete.value = backupID
  showDeleteModal.value = true
  loading.start('backup', backupID, 'delete')

  // Wait for the user to confirm deletion in the modal
}

function cancelDeleteBackup(backupID: string) {
  showDeleteModal.value = false
  loading.stop('backup', backupID, 'delete')
}

function deleteBackup(backupID: string) {
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
      fetchBackupsWithoutLoading() // Refresh the list after deletion
      toastSuccess(`Backup is now ${protect ? 'protected' : 'unprotected'}.`)
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
      name: name.value.trim(),
      notes: notes.value.trim(),
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
      name.value = ''
      notes.value = ''
    })
}

const truncateLength = 50

function truncateNotes(notes: string) {
  if (notes.length > truncateLength) {
    return notes.substring(0, truncateLength - 3) + '...'
  }
  return notes
}

function openNotesModal(title: string, body: string) {
  notesModalTitle.value = title
  notesModalBody.value = body

  const el = notesModalRef.value
  if (!el) return
  if (!el.open) el.showModal()
}

function closeNotesModal() {
  const el = notesModalRef.value
  if (!el) return
  if (el.open) el.close()
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
    fetchBackupsWithoutLoading()
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
            <th scope="col" class="font-medium uppercase">
              Protected ({{ numberOfProtectedBackups() }}/{{ maxProtectedBackupsForUser }})
            </th>
            <th scope="col" class="font-medium uppercase">
              <div class="flex justify-center">Actions</div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="bk in backups" :key="bk.name">
            <td>{{ bk.name }}</td>
            <td>{{ formatDate(bk.ctime) }}</td>
            <td class="max-w-96">
              <p class="hover:link" @click="openNotesModal(bk.name, bk.notes)">
                {{ truncateNotes(bk.notes) }}
              </p>
            </td>
            <td class="" :class="getStatusClass(bk.protected.toString())">
              <div
                :class="{
                  'tooltip tooltip-bottom': !bk.protected && haveFinishedProtectedBackups(),
                }"
                data-tip="Max number of protected backups reached"
              >
                <button
                  @click="protectBackup(bk.id, !bk.protected)"
                  :class="bk.protected ? 'btn btn-accent' : 'btn btn-primary'"
                  :disabled="
                    (!bk.protected && haveFinishedProtectedBackups()) ||
                    loading.is('backup', bk.id, 'protect')
                  "
                  class="btn btn-sm md:btn-md btn-outline w-32 rounded-lg"
                >
                  <span
                    v-if="loading.is('backup', bk.id, 'protect')"
                    class="loading loading-spinner loading-xs"
                  ></span>

                  <IconVue
                    v-else
                    :icon="
                      bk.protected ? 'material-symbols:toggle-on' : 'material-symbols:toggle-off'
                    "
                    class="text-lg"
                  />
                  <p class="hidden md:inline">{{ bk.protected ? 'Unprotect' : 'Protect' }}</p>
                </button>
              </div>
            </td>
            <td
              v-if="$props.vm.group_role !== 'member'"
              class="flex justify-evenly gap-2 text-right text-sm font-medium"
            >
              <div
                :class="{ 'tooltip tooltip-left': $props.vm.status != 'stopped' }"
                data-tip="Canot restore if VM is not stopped"
              >
                <button
                  :disabled="$props.vm.status != 'stopped'"
                  @click="preRestoreBackup(bk.id)"
                  class="btn btn-warning btn-outline w-32 rounded-lg"
                >
                  <span
                    v-if="loading.is('backup', bk.id, 'restore')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:settings-backup-restore" class="text-lg" />

                  Restore
                </button>
              </div>
              <div
                :class="{ 'tooltip tooltip-left': !bk.can_delete }"
                data-tip="Cannot delete automatic or protected backups"
              >
                <button
                  @click="preDeleteBackup(bk.id)"
                  class="btn btn-error btn-outline w-32 rounded-lg"
                  :disabled="!bk.can_delete || loading.is('backup', bk.id, 'delete')"
                >
                  <span
                    v-if="loading.is('backup', bk.id, 'delete')"
                    class="loading loading-spinner loading-xs"
                  ></span>

                  <IconVue v-else icon="material-symbols:delete" class="text-lg" />
                  Delete
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Notes modal -->
    <dialog ref="notesModalRef" class="modal modal-bottom sm:modal-middle">
      <div class="modal-box">
        <h3 class="mb-4 text-xl font-bold">Notes for "{{ notesModalTitle }}"</h3>
        <p class="text-balance">{{ notesModalBody }}</p>

        <div class="modal-action">
          <button class="btn" type="button" @click="closeNotesModal()">Close</button>
        </div>
      </div>

      <form method="dialog" class="modal-backdrop">
        <button aria-label="Close"></button>
      </form>
    </dialog>
    <!-- End Notes modal -->

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete Backup"
      positiveText="Delete Backup"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteBackup(backupToDelete!)"
      @negative="cancelDeleteBackup(backupToDelete!)"
    >
      <p>Are you sure you want to delete this backup? This action cannot be undone.</p>
    </ModalAlert>
    <!--- End Delete modal -->

    <!-- Restore modal -->
    <ModalAlert
      :model-value="showRestoreModal"
      title="Restore Backup"
      positiveText="Restore Backup"
      negativeText="Cancel action"
      positiveBtnClass="btn-warning"
      @positive="restoreBackup(backupToRestore!)"
      @negative="cancelRestoreBackup(backupToRestore!)"
    >
      <p>Are you sure you want to restore this backup? This will overwrite the current VM state</p>
    </ModalAlert>
    <!--- End Restore modal -->
  </div>
</template>
