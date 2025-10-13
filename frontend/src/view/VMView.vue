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
      error.value = 'Failed to create VM: ' + err.response.data
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
  <div class="flex flex-col gap-2 p-2">
    <h1 class="flex items-center gap-2 text-3xl font-bold">
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
          class="input w-24 w-full rounded-lg border p-2"
        />
      </div>
      <div>
        <label for="cores">CPU Cores</label>
        <input
          type="number"
          id="cores"
          v-model="cores"
          class="input w-24 w-full rounded-lg border p-2"
        />
      </div>
      <div>
        <label for="ram">RAM (MB)</label>
        <input
          type="number"
          id="ram"
          v-model="ram"
          class="input w-24 w-full rounded-lg border p-2"
        />
      </div>
      <div>
        <label for="disk">Disk (GB)</label>
        <input type="number" id="disk" v-model="disk" class="input w-24 w-full rounded-lg border" />
      </div>
      <div class="flex w-full items-center justify-between">
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

      <div class="flex w-full flex-col">
        <label for="cores">Notes</label>
        <textarea class="textarea w-full" placeholder="VM Notes" v-model="notes"></textarea>
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
          <td class="font-semibold capitalize" :class="getStatusClass(vm.status)">
            {{ vm.status }}
          </td>
          <td>
            <div v-if="vm.notes" class="bg-base-100 w-max rounded-lg p-2 text-xs text-pretty">
              {{ vm.notes }}
            </div>
          </td>

          <td>
            <div class="grid max-w-3/5 grid-cols-2 gap-2 2xl:grid-cols-3">
              <div class="*:btn-sm col-span-2 grid grid-cols-3 items-center gap-2">
                <button
                  v-if="vm.status === 'stopped'"
                  @click="startVM(vm.id)"
                  class="btn btn-success btn-outline col-span-2 rounded-lg"
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
              <div v-show="vm.status !== 'unknown'" class="flex items-center gap-2">
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
