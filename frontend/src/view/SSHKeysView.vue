<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { SSHKey } from '@/types'
import { api } from '@/lib/api'

const keys = ref<SSHKey[]>([])
const name = ref('')
const key = ref('')

function fetchSSHKeys() {
  api
    .get('/ssh-keys')
    .then((res) => {
      keys.value = res.data as SSHKey[]
    })
    .catch((err) => {
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
    <div>This is the SSH keys view for <b>sasso</b>!</div>
    <div class="flex gap-2 items-center">
      <label for="name">Name:</label>
      <input type="text" id="name" v-model="name" class="border p-2 rounded-lg w-48" />
      <label for="key">Key:</label>
      <input type="text" id="key" v-model="key" class="border p-2 rounded-lg w-96" />
      <button class="btn btn-success rounded-lg" @click="addSSHKey()">
        Add Key
      </button>
    </div>

    <div class="overflow-x-auto">
      <table class="table min-w-full divide-y divide-gray-200">
        <thead class="">
          <tr>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
            >
              ID
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
            >
              Name
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider"
            >
              Key
            </th>
            <th scope="col" class="relative px-6 py-3">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="sshKey in keys" :key="sshKey.id">
            <td class="px-6 py-4 whitespace-nowrap">{{ sshKey.id }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ sshKey.name }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ sshKey.key }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
              <button
                @click="deleteSSHKey(sshKey.id)"
                class="bg-red-400 p-2 rounded-lg hover:bg-red-300"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
