<script setup lang="ts">
import { onMounted, ref, onBeforeUnmount, computed } from 'vue'
import { useLoadingStore } from '@/stores/loading'
import CreateNew from '@/components/CreateNew.vue'
import type { VM, Group } from '@/types'
import { api } from '@/lib/api'
import { formatDate, isVMExpired } from '@/lib/utils'
import { getStatusClass } from '@/const'
import BubbleAlert from '@/components/BubbleAlert.vue'

const vms = ref<VM[]>([])
const name = ref('')
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)
const lifetime = ref(1)
const notes = ref('')
const include_global_ssh_keys = ref(true)
const newVMGroupId = ref<number>()
const error = ref('')

const showIds = ref(false)

const groups = ref<Group[]>([])

const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

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

interface VMCreationBody {
  name: string
  cores: number
  ram: number
  disk: number
  lifetime: number
  include_global_ssh_keys: boolean
  notes: string
  group_id?: number
}

function createVM() {
  const body: VMCreationBody = {
    name: name.value,
    cores: cores.value,
    ram: ram.value,
    disk: disk.value,
    lifetime: lifetime.value,
    include_global_ssh_keys: include_global_ssh_keys.value,
    notes: notes.value,
  }
  if (newVMGroupId.value) {
    body.group_id = newVMGroupId.value
  }
  api
    .post('/vm', body)
    .then(() => {
      fetchVMs()
      name.value = ''
      cores.value = 1
      ram.value = 1024
      disk.value = 4
      lifetime.value = 1
      notes.value = ''
      include_global_ssh_keys.value = true
      error.value = ''
      newVMGroupId.value = undefined
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
  loading.start('vm', vmid, 'start')
  api
    .post(`/vm/${vmid}/start`)
    .then(() => fetchVMs())
    .catch((err) => console.error('Failed to start VM:', err))
    .finally(() => loading.stop('vm', vmid, 'start'))
}

function stopVM(vmid: number) {
  loading.start('vm', vmid, 'stop')
  api
    .post(`/vm/${vmid}/stop`)
    .then(() => fetchVMs())
    .catch((err) => console.error('Failed to stop VM:', err))
    .finally(() => loading.stop('vm', vmid, 'stop'))
}

function restartVM(vmid: number) {
  loading.start('vm', vmid, 'restart')
  api
    .post(`/vm/${vmid}/restart`)
    .then(() => fetchVMs())
    .catch((err) => console.error('Failed to restart VM:', err))
    .finally(() => loading.stop('vm', vmid, 'restart'))
}

function fetchGroups() {
  api.get('/groups').then((res) => {
    groups.value = res.data as Group[]
  })
}

let intervalId: number | null = null

onMounted(() => {
  fetchVMs()
  fetchGroups()
  intervalId = setInterval(() => {
    fetchVMs()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
  loading.clear('vm')
})

const nonMemberGroups = computed(() => {
  return groups.value.filter((group) => group.role !== 'member')
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
          class="input w-full rounded-lg border p-2"
          placeholder="My VM Name"
        />
      </div>
      <div>
        <label for="cores">CPU Cores</label>
        <input
          type="number"
          id="cores"
          v-model="cores"
          class="input w-full rounded-lg border p-2"
        />
      </div>
      <div>
        <label for="ram">RAM (MB)</label>
        <input type="number" id="ram" v-model="ram" class="input w-full rounded-lg border p-2" />
      </div>
      <div>
        <label for="disk">Disk (GB)</label>
        <input type="number" id="disk" v-model="disk" class="input w-full rounded-lg border" />
      </div>
      <div>
        <label for="lifetime">Lifetime</label>
        <select class="select w-full rounded-lg border" v-model.number="lifetime">
          <option value="1" selected>1 Month</option>
          <option value="3">3 Months</option>
          <option value="6">6 Months</option>
          <option value="12">12 Months</option>
        </select>
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

      <label for="group">Group (Optional)</label>
      <select v-model="newVMGroupId" class="select select-bordered">
        <option :value="undefined">Me</option>
        <option v-for="group in nonMemberGroups" :key="group.id" :value="group.id">
          {{ group.name }}
        </option>
      </select>
    </CreateNew>

    <table class="table table-auto divide-y">
      <thead>
        <tr>
          <th v-show="showIds" scope="col">ID</th>
          <th scope="col">Name</th>
          <th scope="col">Group</th>
          <th scope="col">Cores</th>
          <th scope="col">RAM (MB)</th>
          <th scope="col">Disk (GB)</th>
          <th scope="col">Status</th>
          <th scope="col" class="w-80">Lifetime</th>
          <th scope="col" class="flex justify-between">
            <div>Actions</div>
            <button class="badge badge-warning rounded-lg" @click="showIds = !showIds">
              <IconVue v-if="showIds" icon="material-symbols:visibility-off" class="text-xs" />
              <IconVue v-else icon="material-symbols:visibility" class="text-xs" />
              {{ showIds ? 'Hide' : 'Show' }} IDs
            </button>
          </th>
        </tr>
      </thead>
      <tbody class="divide-y">
        <tr v-for="vm in vms" :key="vm.id">
          <Transition>
            <td v-show="showIds">
              {{ vm.id }}
            </td>
          </Transition>
          <td class="min-w-40 text-lg font-semibold">{{ vm.name }}</td>
          <td class="">{{ vm.group_name ? vm.group_name : 'Me' }}</td>
          <td class="">{{ vm.cores }}</td>
          <td class="">{{ vm.ram }}</td>
          <td class="">{{ vm.disk }}</td>
          <td class="font-semibold capitalize" :class="getStatusClass(vm.status)">
            {{ vm.status }}
          </td>
          <td>{{ formatDate(vm.lifetime) }}</td>

          <td>
            <div class="grid grid-cols-2 gap-2">
              <div class="*:btn-sm col-span-2 grid grid-cols-3 items-center gap-2 xl:col-span-1">
                <button
                  v-if="vm.status === 'stopped'"
                  @click="startVM(vm.id)"
                  :disabled="
                    isLoading(vm.id, 'start') ||
                    isVMExpired(vm.lifetime) ||
                    vm.group_role == 'member'
                  "
                  class="btn btn-success btn-outline col-span-2 rounded-lg"
                >
                  <span
                    v-if="isLoading(vm.id, 'start')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:play-arrow" class="text-lg" />
                  <span class="hidden lg:inline">Start</span>
                </button>

                <button
                  v-if="vm.status === 'running'"
                  @click="stopVM(vm.id)"
                  :disabled="isLoading(vm.id, 'stop') || vm.group_role == 'member'"
                  class="btn btn-warning btn-outline rounded-lg"
                >
                  <span
                    v-if="isLoading(vm.id, 'stop')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:stop" class="text-lg" />
                  <span class="hidden lg:inline">Stop</span>
                </button>

                <button
                  v-if="vm.status === 'running'"
                  @click="restartVM(vm.id)"
                  :disabled="isLoading(vm.id, 'restart') || vm.group_role == 'member'"
                  class="btn btn-info btn-outline rounded-lg"
                >
                  <span
                    v-if="isLoading(vm.id, 'restart')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="codicon:debug-restart" class="text-lg" />
                  <span class="hidden lg:inline">Restart</span>
                </button>

                <button
                  v-if="vm.status === 'unknown'"
                  @click="deleteVM(vm.id)"
                  :disabled="vm.group_role == 'member'"
                  class="btn btn-error btn-outline col-span-2 rounded-lg"
                >
                  <IconVue icon="material-symbols:delete" class="text-lg" />
                  <span class="hidden lg:inline">Delete</span>
                </button>
              </div>
              <div>
                <RouterLink
                  v-if="vm.status !== 'pre-deleting' && vm.status !== 'deleting'"
                  :to="`/vm/${vm.id}`"
                  class="btn btn-primary btn-sm md:btn-md btn-outline rounded-lg"
                >
                  <IconVue icon="material-symbols:edit" class="text-lg" />
                  <p class="hidden md:inline">Manage</p>
                </RouterLink>
              </div>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
