<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { SSHKey } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'

const keys = ref<SSHKey[]>([])
const newKey = ref<{ name: string; key: string }>({ name: '', key: '' })

async function getKeys() {
  try {
    const res = await api.get('/admin/ssh-keys/global')
    keys.value = res.data as SSHKey[]
  } catch (error) {
    console.error('Error fetching keys:', error)
    keys.value = []
  }
}

async function addKey() {
  try {
    const res = await api.post('/admin/ssh-keys/global', newKey.value)
    keys.value.push(res.data)
    newKey.value.name = ''
    newKey.value.key = ''
  } catch (error) {
    console.error('Error adding key:', error)
  }
}

async function deleteKey(id: number) {
  try {
    await api.delete(`/admin/ssh-keys/global/${id}`)
    keys.value = keys.value.filter((key) => key.id !== id)
  } catch (error) {
    console.error('Error deleting key:', error)
  }
}

onMounted(getKeys)
</script>

<template>
  <div class="p-4 sm:p-6 lg:p-8">
    <AdminBreadcrumbs />
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
      <div class="inline-block min-w-full py-2 align-middle">
        <div class="border-primary border-opacity-10 rounded-lg border px-2">
          <table class="table min-w-full divide-y">
            <thead class="">
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Name</th>
                <th scope="col">Key</th>
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
                <td class="whitespace-nowrap">
                  {{ key.key }}
                </td>
                <td>
                  <button @click="deleteKey(key.id)" class="btn btn-error">Delete</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
