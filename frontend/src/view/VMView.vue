<script setup lang="ts">
import { onMounted, ref } from 'vue'
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

setInterval(() => {
  // FIXME: timer infinito
  fetchVMs()
}, 5000)

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

onMounted(() => {
  fetchVMs()
})

let openCreate = ref(false);
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <h1 class="text-3xl font-bold flex items-center gap-2">
      <IconVue class="text-primary" icon="mi:computer"></IconVue>Virtual Machine
    </h1>

    <div>
      <button class="btn btn-primary rounded-xl" @click="openCreate = !openCreate">
        <IconVue icon="mi:add" class="text-xl"></IconVue>{{openCreate? 'Close':'Create VM'}}
      </button>
    </div>
    <div v-if="openCreate">
      <div class="p-4 border border-primary rounded-xl bg-base-200 flex flex-col gap-4 w-full h-full">
        <div>
          <label for="cores">CPU Cores:</label>
          <input type="number" id="cores" v-model="cores" class="input w-full border p-2 rounded-lg w-24" />
        </div>
        <div>
          <label for="ram">RAM (MB):</label>
          <input type="number" id="ram" v-model="ram" class="input w-full border p-2 rounded-lg w-24" />
        </div>
        <div>
          <label for="disk">Disk (GB):</label>
          <input type="number" id="disk" v-model="disk" class="input w-full border rounded-lg w-24" />
        </div>
        <div class="flex items-center">
          <input type="checkbox" id="include_global_ssh_keys" v-model="include_global_ssh_keys"
            class="checkbox checkbox-primary" />
          <label for="include_global_ssh_keys" class="ml-2">Include Global SSH Keys</label>
        </div>
        <button class="btn btn-success p-2 rounded-lg" @click="createVM()">Create VM</button>

      </div>
    </div>

    <div class="alert alert-info p-4" role="alert">
      <p class="font-bold">Information</p>
      <p>
        Including the global SSH keys will allow for better troubleshooting if something is not
        working.
      </p>
    </div>
    <div class="alert alert-warning p-4" role="alert">
      <p class="font-bold">Warning</p>
      <p>
        Stopping and restarting VMs is like pulling the power cord, it is not a graceful shutdown.
        Use with caution and only if necessary (like when the VM is not responding).
      </p>
    </div>

    <div class="overflow-x-auto">
      <table class="table min-w-full *:text-base-content">
        <thead>
          <tr>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              ID
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Cores
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              RAM (MB)
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Disk (GB)
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th scope="col" class="relative px-6 py-3">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="vm in vms" :key="vm.id">
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.id }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.cores }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.ram }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.disk }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ vm.status }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium flex gap-2">
              <RouterLink :to="`/vm/${vm.id}/interfaces`" class="btn btn-primary mr-4">Interfaces</RouterLink>
              <button v-if="vm.status === 'stopped'" @click="startVM(vm.id)"
                class="btn btn-success rounded-lg btn-outline">
                Start
              </button>
              <button v-if="vm.status === 'running'" @click="stopVM(vm.id)"
                class="btn btn-warning rounded-lg btn-outline">
                Stop
              </button>
              <button v-if="vm.status === 'running'" @click="restartVM(vm.id)"
                class="btn btn-info rounded-lg btn-outline">
                Restart
              </button>
              <button @click="deleteVM(vm.id)" class="btn btn-error rounded-lg btn-outline">
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
