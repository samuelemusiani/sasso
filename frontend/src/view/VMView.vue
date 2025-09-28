<script setup lang="ts">
import { onMounted, ref, onBeforeUnmount } from 'vue'
import type { VM } from '@/types'
import { api } from '@/lib/api'

const vms = ref<VM[]>([])
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)
const include_global_ssh_keys = ref(true)

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

function createVM() {
  api
    .post('/vm', {
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
      include_global_ssh_keys: include_global_ssh_keys.value,
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

function startVM(vmid: number) {
  api
    .post(`/vm/${vmid}/start`)
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to start VM:', err)
    })
}

function stopVM(vmid: number) {
  api
    .post(`/vm/${vmid}/stop`)
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to stop VM:', err)
    })
}

function restartVM(vmid: number) {
  api
    .post(`/vm/${vmid}/restart`)
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to restart VM:', err)
    })
}

let intervalId: number | null = null

onMounted(() => {
  fetchVMs()
  intervalId = setInterval(() => {
    fetchVMs()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
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
      <div class="flex items-center">
        <input
          type="checkbox"
          id="include_global_ssh_keys"
          v-model="include_global_ssh_keys"
          class="border p-2 rounded-lg"
        />
        <label for="include_global_ssh_keys" class="ml-2">Include Global SSH Keys</label>
      </div>
      <button class="bg-green-400 p-2 rounded-lg hover:bg-green-300" @click="createVM()">
        Create VM
      </button>
    </div>
    <div class="bg-blue-100 border-l-4 border-blue-500 text-blue-700 p-4" role="alert">
      <p class="font-bold">Information</p>
      <p>
        Including the global SSH keys will allow for better troubleshooting if something is not
        working.
      </p>
    </div>
    <div class="bg-yellow-100 border-l-4 border-yellow-500 text-yellow-700 p-4" role="alert">
      <p class="font-bold">Warning</p>
      <p>
        Stopping and restarting VMs is like pulling the power cord, it is not a graceful shutdown.
        Use with caution and only if necessary (like when the VM is not responding).
      </p>
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
              <RouterLink
                :to="`/vm/${vm.id}/interfaces`"
                class="text-indigo-600 hover:text-indigo-900 mr-4"
                >Interfaces
              </RouterLink>
              <RouterLink
                :to="`/vm/${vm.id}/backups`"
                class="text-orange-600 hover:text-orange-900 mr-4"
                >Backups
              </RouterLink>
              <button
                v-if="vm.status === 'stopped'"
                @click="startVM(vm.id)"
                class="bg-green-400 p-2 rounded-lg hover:bg-green-300 mr-2"
              >
                Start
              </button>
              <button
                v-if="vm.status === 'running'"
                @click="stopVM(vm.id)"
                class="bg-yellow-400 p-2 rounded-lg hover:bg-yellow-300 mr-2"
              >
                Stop
              </button>
              <button
                v-if="vm.status === 'running'"
                @click="restartVM(vm.id)"
                class="bg-blue-400 p-2 rounded-lg hover:bg-blue-300 mr-2"
              >
                Restart
              </button>
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
