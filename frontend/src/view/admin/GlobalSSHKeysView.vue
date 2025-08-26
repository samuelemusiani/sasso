<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { SSHKey } from '@/types'

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
    <div class="sm:flex sm:items-center">
      <div class="sm:flex-auto">
        <h1 class="text-2xl font-bold leading-6 text-gray-900">Global SSH Keys</h1>
        <p class="mt-2 text-sm text-gray-700">A list of all the global SSH keys in the system.</p>
      </div>
    </div>
    <div class="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <div class="flow-root">
          <div class="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
            <div class="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
              <div class="overflow-hidden shadow ring-1 ring-black ring-opacity-5 sm:rounded-lg">
                <table class="min-w-full divide-y divide-gray-300">
                  <thead class="bg-gray-50">
                    <tr>
                      <th
                        scope="col"
                        class="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6"
                      >
                        ID
                      </th>
                      <th
                        scope="col"
                        class="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
                      >
                        Name
                      </th>
                      <th scope="col" class="relative py-3.5 pl-3 pr-4 sm:pr-6">
                        <span class="sr-only">Delete</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-200 bg-white">
                    <tr v-if="keys.length === 0">
                      <td
                        colspan="3"
                        class="whitespace-nowrap px-3 py-4 text-sm text-gray-500 text-center"
                      >
                        No keys found.
                      </td>
                    </tr>
                    <tr v-for="key in keys" :key="key.id">
                      <td
                        class="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6"
                      >
                        {{ key.id }}
                      </td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
                        {{ key.name }}
                      </td>
                      <td
                        class="relative whitespace-nowrap py-4 pl-3 pr-4 text-right text-sm font-medium sm:pr-6"
                      >
                        <button @click="deleteKey(key.id)" class="text-red-600 hover:text-red-900">
                          Delete
                        </button>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="lg:col-span-1">
        <div class="bg-white shadow sm:rounded-lg">
          <div class="px-4 py-5 sm:p-6">
            <h3 class="text-lg font-medium leading-6 text-gray-900">Add a new key</h3>
            <form @submit.prevent="addKey" class="mt-5 space-y-4">
              <div>
                <label for="name" class="block text-sm font-medium text-gray-700"> Name </label>
                <div class="mt-1">
                  <input
                    v-model="newKey.name"
                    type="text"
                    name="name"
                    id="name"
                    class="block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
                    placeholder="My awesome key"
                  />
                </div>
              </div>
              <div>
                <label for="key" class="block text-sm font-medium text-gray-700"> Key </label>
                <div class="mt-1">
                  <textarea
                    v-model="newKey.key"
                    id="key"
                    name="key"
                    rows="4"
                    class="block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
                    placeholder="ssh-rsa AAAA..."
                  ></textarea>
                </div>
              </div>
              <div>
                <button
                  type="submit"
                  class="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                >
                  Add Key
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
