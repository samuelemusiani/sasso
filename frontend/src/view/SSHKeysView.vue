<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { SSHKey } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'

const keys = ref<SSHKey[]>([])
const name = ref('')
const key = ref('')
const error = ref('')

function fetchSSHKeys() {
  api
    .get('/ssh-keys')
    .then((res) => {
      keys.value = res.data as SSHKey[]
    })
    .catch((err) => {
      error.value = 'Failed to fetch SSH keys: ' + err.response.data
      console.error('Failed to fetch SSH keys:', err)
    })
}

function addSSHKey() {
  api
    .post('/ssh-keys', {
      name: name.value,
      key: key.value,
    })
    .then(() => {
      fetchSSHKeys()
      name.value = ''
      key.value = ''
    })
    .catch((err) => {
      console.log('Error details:', err.response.data)
      error.value = 'Failed to add SSH key: ' + err.response.data
      console.error('Failed to add SSH key:', err)
    })
}

function deleteSSHKey(id: number) {
  if (confirm('Are you sure you want to delete this SSH key?')) {
    api
      .delete(`/ssh-keys/${id}`)
      .then(() => {
        fetchSSHKeys()
      })
      .catch((err) => {
        error.value = 'Failed to delete SSH key: ' + err.response.data
        console.error('Failed to delete SSH key:', err)
      })
  }
}

onMounted(() => {
  fetchSSHKeys()
})
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <h1 class="text-3xl font-bold text-base-content flex items-center gap-2">
      <IconVue icon="material-symbols:key" class="text-primary" />
      SSH Keys
    </h1>
    <CreateNew title="SSH Key" :create="addSSHKey" :error="error">
      <div class="flex flex-col gap-2">
        <label for="name">Name</label>
        <input
          v-model="name"
          type="text"
          placeholder="Key Name"
          class="input w-full p-2 border border-primary rounded-lg"
        />

        <label for="key">Key</label>
        <input
          v-model="key"
          type="text"
          placeholder="SSH Public Key"
          class="input w-full p-2 border border-primary rounded-lg"
        />
      </div>
    </CreateNew>
    <div class="overflow-x-auto">
      <table class="table min-w-full divide-y divide-gray-200">
        <thead class="">
          <tr>
            <th scope="col" class="">Name</th>
            <th scope="col" class="">Key</th>
            <th scope="col" class="relative px-6 py-3">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="sshKey in keys" :key="sshKey.id">
            <td class="whitespace-nowrap">{{ sshKey.name }}</td>
            <td class="whitespace-nowrap">{{ sshKey.key }}</td>
            <td class="text-right text-sm font-medium">
              <button
                @click="deleteSSHKey(sshKey.id)"
                class="btn btn-error rounded-lg btn-sm md:btn-md btn-outline"
              >
                <IconVue icon="material-symbols:delete" class="text-lg"></IconVue>
                <p class="hidden md:inline">Delete</p>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
