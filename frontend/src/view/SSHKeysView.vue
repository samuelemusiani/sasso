<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { SSHKey } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import ModalAlert from '@/components/ModalAlert.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()

const loading = useLoadingStore()

const keys = ref<SSHKey[]>([])
const name = ref('')
const key = ref('')
const error = ref('')

const wrap = ref(false)
function toggleWrap() {
  wrap.value = !wrap.value
}

function fetchSSHKeys() {
  loading.start('sshKeys', null, 'fetch')
  api
    .get('/ssh-keys')
    .then((res) => {
      keys.value = res.data as SSHKey[]
    })
    .catch((err) => {
      console.error('Failed to fetch SSH keys:', err)
      toastError('Failed to fetch SSH keys: ' + err.response)
    })
    .finally(() => {
      loading.stop('sshKeys', null, 'fetch')
    })
}

function addSSHKey() {
  return api
    .post('/ssh-keys', {
      name: name.value,
      key: key.value,
    })
    .then(() => {
      fetchSSHKeys()
      name.value = ''
      key.value = ''
      return true
    })
    .catch((err) => {
      console.log('Error details:', err.response.data)
      error.value = 'Failed to add SSH key: ' + err.response.data
      console.error('Failed to add SSH key:', err)
      return false
    })
}

const showDeleteModal = ref(false)
const sshKeyToDelete = ref<number | null>(null)

function preDeleteSSHKey(id: number) {
  sshKeyToDelete.value = id
  showDeleteModal.value = true
  loading.start('sshKey', id, 'delete')
}

function deleteSSHKey(id: number) {
  api
    .delete(`/ssh-keys/${id}`)
    .then(() => {
      fetchSSHKeys()
    })
    .catch((err) => {
      console.error('Failed to delete SSH key:', err)
      toastError('Failed to delete SSH key')
    })
    .finally(() => {
      sshKeyToDelete.value = null
      showDeleteModal.value = false
      loading.stop('sshKey', id, 'delete')
    })
}

function cancelDeleteSSHKey(id: number) {
  sshKeyToDelete.value = null
  showDeleteModal.value = false
  loading.stop('sshKey', id, 'delete')
}

onMounted(() => {
  fetchSSHKeys()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="text-base-content flex items-center gap-2 text-3xl font-bold">
        <IconVue icon="material-symbols:key" class="text-primary" />
        SSH Keys
      </h1>
      <HelpButton />
    </div>
    <CreateNew title="SSH Key" :create="addSSHKey" :error="error" :close-on-create="true">
      <div class="flex flex-col gap-2">
        <label for="name">Name</label>
        <input
          v-model="name"
          type="text"
          placeholder="Key Name"
          class="input border-primary w-full rounded-lg border p-2"
        />

        <label for="key">Key</label>
        <input
          v-model="key"
          type="text"
          placeholder="SSH Public Key"
          class="input border-primary w-full rounded-lg border p-2"
        />
      </div>
    </CreateNew>

    <div v-if="loading.is('sshKeys', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <div v-else>
      <table class="table min-w-full divide-y divide-gray-200">
        <thead class="">
          <tr>
            <th scope="col" class="">Name</th>
            <th scope="col" class="">Key</th>
            <th scope="col" class="">
              <div class="flex items-center justify-between gap-2">
                <div>Actions</div>
                <button class="btn btn-sm btn-warning w-28 rounded-lg" @click="toggleWrap">
                  <IconVue :icon="wrap ? 'mdi:unwrap' : 'mdi:wrap'" class="text-lg" />
                  {{ wrap ? 'Unwrap' : 'Wrap' }}
                </button>
              </div>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="sshKey in keys" :key="sshKey.id">
            <td class="text-lg font-semibold whitespace-nowrap">{{ sshKey.name }}</td>
            <td
              class="max-w-80 lg:max-w-104 xl:max-w-136 2xl:max-w-200"
              :class="wrap ? 'break-all' : 'text-nowrap'"
            >
              <div class="overflow-x-auto text-ellipsis">
                {{ sshKey.key }}
              </div>
            </td>
            <td class="text-right text-sm font-medium">
              <button
                @click="preDeleteSSHKey(sshKey.id)"
                class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
                :disabled="loading.is('sshKey', sshKey.id, 'delete')"
              >
                <span
                  v-if="loading.is('sshKey', sshKey.id, 'delete')"
                  class="loading loading-spinner loading-xs"
                ></span>
                <IconVue v-else icon="material-symbols:delete" class="text-lg"></IconVue>
                <p class="hidden md:inline">Delete</p>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete SSH Key"
      positiveText="Delete Key"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteSSHKey(sshKeyToDelete!)"
      @negative="cancelDeleteSSHKey(sshKeyToDelete!)"
    >
      <p>Are you sure you want to delete this SSH key? This action cannot be undone.</p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
