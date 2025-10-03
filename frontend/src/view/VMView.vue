<script setup lang="ts">
import { onMounted, ref, onBeforeUnmount } from 'vue'
import CreateNew from '@/components/CreateNew.vue'
import type { VM } from '@/types'
import { api } from '@/lib/api'
import { getStatusClass } from '@/const'
import BubbleAlert from '@/components/BubbleAlert.vue'

const vms = ref<VM[]>([])
const name = ref('')
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)
const notes = ref('')
const include_global_ssh_keys = ref(true)
const error = ref('')

function fetchVMs() {
  api
    .get('/vm')
    .then((res) => {
      const tmp = res.data.sort((a: VM, b: VM) => a.id - b.id)
      vms.value = tmp as VM[]
    })
    .catch((err) => {
      console.error('Failed to fetch VMs:', err)
    })
}

function createVM() {
  api
    .post('/vm', {
      name: name.value,
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
      include_global_ssh_keys: include_global_ssh_keys.value,
      notes: notes.value,
    })
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to create VM:', err)
      error.value = 'Failed to create VM: ' + err.message
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
    <h1 class="text-3xl font-bold flex items-center gap-2">
      <IconVue class="text-primary" icon="mi:computer"></IconVue>Virtual Machine
    </h1>

    <CreateNew title="New VM" :create="createVM" :error="error">
      <div>
        <label for="cores">Name</label>
        <input
          required
          type="text"
          id="name"
          v-model="name"
          class="input w-full border p-2 rounded-lg w-24"
        />
      </div>
      <div>
        <label for="cores">CPU Cores</label>
        <input
          type="number"
          id="cores"
          v-model="cores"
          class="input w-full border p-2 rounded-lg w-24"
        />
      </div>
      <div>
        <label for="ram">RAM (MB)</label>
        <input
          type="number"
          id="ram"
          v-model="ram"
          class="input w-full border p-2 rounded-lg w-24"
        />
      </div>
      <div>
        <label for="disk">Disk (GB)</label>
        <input type="number" id="disk" v-model="disk" class="input w-full border rounded-lg w-24" />
      </div>
      <div class="flex items-center justify-between w-full">
        <div class="flex items-center gap-2">
          <input
            type="checkbox"
            id="include_global_ssh_keys"
            v-model="include_global_ssh_keys"
            class="checkbox checkbox-primary"
          />
          <label for="include_global_ssh_keys">Include Global SSH Keys</label>
          <BubbleAlert type="info">
            Including the global SSH keys will allow for better troubleshooting if something is not
            working.
          </BubbleAlert>
        </div>
      </div>

      <div class="flex flex-col w-full">
        <label for="cores">Notes</label>
        <textarea class="w-full textarea" placeholder="VM Notes" v-model="notes"></textarea>
      </div>
    </CreateNew>

    <table class="table table-auto divide-y">
      <thead>
        <tr>
          <th scope="col">Name</th>
          <th scope="col">Cores</th>
          <th scope="col">RAM (MB)</th>
          <th scope="col">Disk (GB)</th>
          <th scope="col">Status</th>
          <th scope="col">Notes</th>
          <th scope="col"></th>
        </tr>
      </thead>
      <tbody class="divide-y">
        <tr v-for="vm in vms" :key="vm.id">
          <td class="text-lg">{{ vm.name }}</td>
          <td class="">{{ vm.cores }}</td>
          <td class="">{{ vm.ram }}</td>
          <td class="">{{ vm.disk }}</td>
          <td class="capitalize font-semibold" :class="getStatusClass(vm.status)">
            {{ vm.status }}
          </td>
          <td>
            <div v-if="vm.notes" class="text-xs bg-base-100 p-2 rounded-lg w-max text-pretty">
              {{ vm.notes }}
            </div>
          </td>

          <td>
            <div class="grid grid-cols-2 2xl:grid-cols-3 gap-2 max-w-3/5">
              <div class="grid grid-cols-3 gap-2 col-span-2 *:btn-sm items-center">
                <button
                  v-if="vm.status === 'stopped'"
                  @click="startVM(vm.id)"
                  class="col-span-2 btn btn-success btn-outline rounded-lg"
                >
                  <IconVue icon="material-symbols:play-arrow" class="text-lg" />
                  <span class="hidden md:inline">Start</span>
                </button>

                <button
                  v-if="vm.status === 'running'"
                  @click="stopVM(vm.id)"
                  class="btn btn-warning btn-outline rounded-lg"
                >
                  <IconVue icon="material-symbols:stop" class="text-lg" />
                  <span class="hidden md:inline">Stop</span>
                </button>

                <button
                  v-if="vm.status === 'running'"
                  @click="restartVM(vm.id)"
                  class="btn btn-info btn-outline rounded-lg"
                >
                  <IconVue icon="codicon:debug-restart" class="text-lg" />
                  <span class="hidden md:inline">Restart</span>
                </button>

                <button @click="deleteVM(vm.id)" class="btn btn-error btn-outline rounded-lg">
                  <IconVue icon="material-symbols:delete" class="text-lg" />
                  <span class="hidden md:inline">Delete</span>
                </button>
              </div>
              <div class="flex gap-2 items-center">
                <RouterLink
                  :to="`/vm/${vm.id}/interfaces`"
                  class="btn btn-primary btn-sm md:btn-md rounded-lg"
                >
                  <IconVue icon="material-symbols:network-node" class="text-lg" />
                  <span class="hidden md:inline">Interfaces</span>
                </RouterLink>

                <RouterLink
                  :to="`/vm/${vm.id}/backups`"
                  class="btn btn-secondary btn-sm md:btn-md rounded-lg"
                >
                  <IconVue icon="material-symbols:backup" class="text-lg" />
                  <span class="hidden md:inline">Backup</span>
                </RouterLink>
              </div>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
