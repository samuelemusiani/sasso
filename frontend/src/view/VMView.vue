<script setup lang="ts">
import { onMounted, ref, onBeforeUnmount } from 'vue'
import CreateNew from '@/components/CreateNew.vue'
import type { VM } from '@/types'
import { api } from '@/lib/api'

const vms = ref<VM[]>([])
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)
const include_global_ssh_keys = ref(true)
let openCreate = ref(false)

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

let intervalId: number | null = null

// pre-creating, pre-deleting, deleting, creating, running, stopped, suspended, unknown, pre-configuring, configuring

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
  <div class="flex flex-col gap-2 overflow-y-auto">
    <h1 class="text-3xl font-bold flex items-center gap-2">
      <IconVue class="text-primary" icon="mi:computer"></IconVue>Virtual Machine
    </h1>

    <CreateNew title="Create New VM" :create="createVM">
      <div>
        <label for="cores">CPU Cores:</label>
        <input type="number" id="cores" v-model="cores" class="input w-full border p-2 rounded-lg w-24" />
      </div>
      <div>
        <label for="ram">RAM (MB):</label>
        <input type="number" id="ram" v-model="ram" class="input w-full border p-2 rounded-lg w-24" />
      </div>
      <div>
        <label for="disk">Disk (GB)</label>
        <input type="number" id="disk" v-model="disk" class="input w-full border rounded-lg w-24" />
      </div>
      <div class="flex items-center">
        <input type="checkbox" id="include_global_ssh_keys" v-model="include_global_ssh_keys"
          class="checkbox checkbox-primary" />
        <label for="include_global_ssh_keys" class="ml-2">Include Global SSH Keys</label>
      </div>
    </CreateNew>

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
      <table class="table">
        <thead>
          <tr>
            <th scope="col" class="text-xs uppercase">
              ID
            </th>
            <th scope="col" class="text-xs uppercase">
              Cores
            </th>
            <th scope="col" class="text-xs uppercase">
              RAM (MB)
            </th>
            <th scope="col" class="text-xs uppercase">
              Disk (GB)
            </th>
            <th scope="col" class="text-xs uppercase">
              Status
            </th>
            <th scope="col" class="relative">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="vm in vms" :key="vm.id">
            <td class="">{{ vm.id }}</td>
            <td class="">{{ vm.cores }}</td>
            <td class="">{{ vm.ram }}</td>
            <td class="">{{ vm.disk }}</td>
            <td class="">{{ vm.status }}</td>
            <td class=" flex gap-2 items-center">
              <RouterLink :to="`/vm/${vm.id}/interfaces`" class="btn btn-primary btn-sm md:btn-md rounded-lg">
                <IconVue icon="material-symbols:network-node" class="text-lg" />
                <p class="hidden md:inline">Interfaces</p>
              </RouterLink>
              <RouterLink :to="`/vm/${vm.id}/backups`" class="btn btn-secondary btn-sm md:btn-md rounded-lg">
                <IconVue icon="material-symbols:backup" class="text-lg" />
                <p class="hidden md:inline">Backup</p>
              </RouterLink>
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
              <button @click="deleteVM(vm.id)" class="btn btn-error rounded-lg btn-sm md:btn-md btn-outline">
                <IconVue icon="material-symbols:delete" class="text-lg" />
                <p class="hidden md:inline">Delete</p>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
