<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { SSHKey } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import ModalAlert from '@/components/ModalAlert.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const keys = ref<SSHKey[]>([])
const newKey = ref<{ name: string; key: string }>({ name: '', key: '' })

const wrap = ref(false)
function toggleWrap() {
  wrap.value = !wrap.value
}

function fetchKeys() {
  loading.start('globalSSHKeys', null, 'fetch')
  api
    .get('/admin/ssh-keys/global')
    .then((res) => {
      keys.value = res.data as SSHKey[]
    })
    .catch((error) => {
      console.error('Error fetching keys:', error)
      toastError('Failed to fetch global SSH keys: ' + error.response.data)
      keys.value = []
    })
    .finally(() => {
      loading.stop('globalSSHKeys', null, 'fetch')
    })
}

function addKey() {
  api
    .post('/admin/ssh-keys/global', newKey.value)
    .then((res) => {
      keys.value.push(res.data)
      newKey.value.name = ''
      newKey.value.key = ''
    })
    .catch((error) => {
      console.error('Error adding key:', error)
      toastError('Failed to add global SSH key: ' + error.response.data)
    })
}

const showDeleteModal = ref(false)
const sshKeyToDelete = ref<number | null>(null)

function preDeleteKey(id: number) {
  sshKeyToDelete.value = id
  showDeleteModal.value = true
  loading.start('globalSSHKeys', id, 'delete')
}

function deleteKey(id: number) {
  api
    .delete(`/admin/ssh-keys/global/${id}`)
    .then(() => {
      keys.value = keys.value.filter((key) => key.id !== id)
    })
    .catch((error) => {
      console.error('Error deleting key:', error)
    })
    .finally(() => {
      showDeleteModal.value = false
      sshKeyToDelete.value = null
      loading.stop('globalSSHKeys', id, 'delete')
    })
}

function cancelDeleteKey(id: number) {
  sshKeyToDelete.value = null
  showDeleteModal.value = false
  loading.stop('globalSSHKeys', id, 'delete')
}

onMounted(() => {
  fetchKeys()
})
</script>

<template>
  <div class="p-2">
    <div class="flex justify-between">
      <AdminBreadcrumbs />
      <HelpButton />
    </div>
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h1 class="text-2xl leading-6 font-bold">Global SSH Keys</h1>
        <p class="mt-2 text-sm">Global SSH keys in the system.</p>
      </div>
    </div>
    <div class="mt-8">
      <div class="border-primary border-opacity-10 rounded-lg border p-4">
        <h3 class="text-lg">Add a new key</h3>
        <form @submit.prevent="addKey" class="mt-5 space-y-4">
          <div>
            <label for="name" class="block text-sm"> Name </label>
            <div class="mt-1">
              <input
                v-model="newKey.name"
                type="text"
                name="name"
                id="name"
                class="input rounded-lg"
                placeholder="My awesome key"
              />
            </div>
          </div>
          <div>
            <label for="key" class="block text-sm"> Key </label>
            <div class="mt-1">
              <textarea
                v-model="newKey.key"
                id="key"
                name="key"
                rows="4"
                class="textarea w-full rounded-lg"
                placeholder="ssh-rsa AAAA..."
              ></textarea>
            </div>
          </div>
          <div>
            <button type="submit" class="btn btn-primary w-full rounded-lg">Add Key</button>
          </div>
        </form>
      </div>
      <div class="border-primary border-opacity-10 my-2 overflow-x-auto rounded-lg border p-2">
        <div v-if="loading.is('globalSSHKeys', null, 'fetch')" class="grid h-64">
          <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
        </div>

        <table v-else class="table min-w-full divide-y">
          <thead class="">
            <tr>
              <th scope="col">ID</th>
              <th scope="col">Name</th>
              <th scope="col">Key</th>
              <th>
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
            <tr v-if="keys.length === 0">
              <td colspan="3" class="text-center">No keys found.</td>
            </tr>
            <tr v-for="key in keys" :key="key.id">
              <td>
                {{ key.id }}
              </td>
              <td>
                {{ key.name }}
              </td>
              <td
                class="max-w-80 lg:max-w-104 xl:max-w-136 2xl:max-w-200"
                :class="wrap ? 'break-all' : 'text-nowrap'"
              >
                <div class="overflow-x-auto text-ellipsis">
                  {{ key.key }}
                </div>
              </td>
              <td>
                <button
                  @click="preDeleteKey(key.id)"
                  class="btn btn-error btn-sm md:btn-md rounded-lg"
                  :disabled="loading.is('globalSSHKeys', key.id, 'delete')"
                >
                  <span
                    v-if="loading.is('globalSSHKeys', key.id, 'delete')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:delete" class="text-lg"></IconVue>
                  <p class="hidden md:inline">Delete</p>
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Delete modal -->
        <ModalAlert
          :model-value="showDeleteModal"
          title="Delete Key"
          positiveText="Delete Key"
          negativeText="Cancel action"
          positiveBtnClass="btn-error"
          @positive="deleteKey(sshKeyToDelete!)"
          @negative="cancelDeleteKey(sshKeyToDelete!)"
        >
          <p>Are you sure you want to delete this Key? This action cannot be undone.</p>
        </ModalAlert>
        <!-- End of Delete modal -->
      </div>
    </div>
  </div>
</template>
