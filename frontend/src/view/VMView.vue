<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { VM } from '@/types'
import { api } from '@/lib/api'

const vms = ref<VM[]>([])
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)

function fetchVMs() {
  api
    .get('/vm')
    .then((res) => {
      vms.value = res.data as VM[]
    })
    .catch((err) => {
      console.error('Failed to fetch VMs:', err)
    })
}

setInterval(() => {
  fetchVMs()
}, 5000)

function createVM() {
  api
    .post('/vm', {
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
    })
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to create VM:', err)
    })
}

function deleteVM(vmid: number) {
  if (confirm(`Are you sure you want to delete VM ${vmid}?`)) {
    api
      .delete(`/vm/${vmid}`)
      .then(() => {
        fetchVMs()
      })
      .catch((err) => {
        console.error('Failed to delete VM:', err)
      })
  }
}

onMounted(() => {
  fetchVMs()
})
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <div>This is the VM view for <b>sasso</b>!</div>
    <div class="flex gap-2 items-center">
      <label for="cores">Cores:</label>
      <input type="number" id="cores" v-model="cores" class="border p-2 rounded-lg w-24" />
      <label for="ram">RAM (MB):</label>
      <input type="number" id="ram" v-model="ram" class="border p-2 rounded-lg w-24" />
      <label for="disk">Disk (GB):</label>
      <input type="number" id="disk" v-model="disk" class="border p-2 rounded-lg w-24" />
      <button class="bg-green-400 p-2 rounded-lg hover:bg-green-300" @click="createVM()">
        Create VM
      </button>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              ID
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Cores
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              RAM (MB)
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Disk (GB)
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Status
            </th>
            <th scope="col" class="relative px-6 py-3">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="vm in vms" :key="vm.id">
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.id }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.cores }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.ram }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.disk }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.status }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
              <button @click="deleteVM(vm.id)" class="bg-red-400 p-2 rounded-lg hover:bg-red-300">
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
