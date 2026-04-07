<script setup lang="ts">
import { onMounted, ref, onBeforeUnmount, computed } from 'vue'
import { useLoadingStore } from '@/stores/loading'
import CreateNew from '@/components/CreateNew.vue'
import type { VM, Group, Template } from '@/types'
import { api } from '@/lib/api'
import { formatDate, isVMExpired } from '@/lib/utils'
import { getStatusClass } from '@/const'
import BubbleAlert from '@/components/BubbleAlert.vue'
import VMStartChecksModal from '@/components/vm/VMStartChecksModal.vue'
import { useVmStartWithChecks } from '@/composables/useVMStartWithChecks'
import { useUserResources } from '@/composables/userResources'
import ModalAlert from '@/components/ModalAlert.vue'

const { fetchUserResources, userFreeResources } = useUserResources(api)

const vms = ref<VM[]>([])
const templates = ref<Template[]>([])
const name = ref('')
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)
const template = ref('')
const lifetime = ref(1)
const notes = ref('')
const include_global_ssh_keys = ref(true)
const newVMGroupId = ref<number>()
const error = ref('')

const showIds = ref(false)

const groups = ref<Group[]>([])

const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

const readyTemplates = computed(() => {
  return templates.value.filter((t) => t.ready)
})

const { showModal, modalMissing, preStartVM, confirmStart, cancelStart } = useVmStartWithChecks({
  api,
  loading,
  onStarted: () => {
    if (startingVMId.value) {
      const vm = vms.value.find((v) => v.id === startingVMId.value)
      if (vm) vm.status = 'running'
    }
    fetchVMs()
  },
})

const startingVMId = ref<number | null>(null)
function preStartVMWrapper(vmid: number) {
  startingVMId.value = vmid
  preStartVM(vmid)
}

const minDiskForCurrentTemplate = computed(() => {
  const selectedTemplate = templates.value.find((t) => t.name === template.value)
  return selectedTemplate ? selectedTemplate.disk : 4
})

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

function fetchTemplates() {
  api
    .get('/vm/templates')
    .then((res) => {
      templates.value = res.data as Template[]

      const selectedStillExists = templates.value.some((t) => t.name === template.value)

      if (!template.value || !selectedStillExists) {
        template.value = templates.value[0]?.name ?? ''
      }
    })
    .catch((err) => {
      console.error('Failed to fetch templates:', err)
    })
}

interface VMCreationBody {
  name: string
  cores: number
  ram: number
  disk: number
  template: string
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
    template: template.value,
    lifetime: lifetime.value,
    include_global_ssh_keys: include_global_ssh_keys.value,
    notes: notes.value,
  }

  if (newVMGroupId.value) {
    body.group_id = newVMGroupId.value
  }

  return api
    .post('/vm', body)
    .then(() => {
      fetchVMs()
      name.value = ''
      cores.value = 1
      ram.value = 1024
      disk.value = 4

      if (templates.value.length > 0) {
        template.value = templates.value[0].name
      } else {
        template.value = ''
      }

      lifetime.value = 1
      notes.value = ''
      include_global_ssh_keys.value = true
      error.value = ''
      newVMGroupId.value = undefined

      return true
    })
    .catch((err) => {
      console.error('Failed to create VM:', err)
      error.value = 'Failed to create VM: ' + err.response.data

      return false
    })
}

const showDeleteModal = ref(false)
const vmToDelete = ref<number | null>(null)

function preDeleteVM(vmid: number) {
  vmToDelete.value = vmid
  showDeleteModal.value = true
  loading.start('vm', vmid, 'delete')
}

function deleteVM(vmid: number) {
  api
    .delete(`/vm/${vmid}`)
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to delete VM:', err)
    })
    .finally(() => {
      vmToDelete.value = null
      showDeleteModal.value = false
      loading.stop('vm', vmid, 'delete')
    })
}

function cancelDeleteVM(vmid: number) {
  vmToDelete.value = null
  showDeleteModal.value = false
  loading.stop('vm', vmid, 'delete')
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
  fetchTemplates()
  fetchGroups()
  fetchUserResources()
  intervalId = setInterval(() => {
    fetchVMs()
    fetchTemplates()
    fetchUserResources()
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
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" icon="mi:computer"></IconVue>Virtual Machine
      </h1>
      <HelpButton />
    </div>

    <CreateNew title="New VM" :create="createVM" :error="error" :close-on-create="true">
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
      <div class="grid grid-cols-3 gap-4">
        <div>
          <label for="cores">CPU Cores</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="cores"
                v-model="cores"
                class="input join-item validator w-full rounded-l-lg border p-2"
                min="1"
                :max="userFreeResources?.free_cpu"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available CPU cores</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_cpu }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div>
          <label for="ram">RAM (MB)</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="ram"
                v-model="ram"
                class="input joint-item validator w-full rounded-l-lg border p-2"
                min="1024"
                :max="userFreeResources?.free_ram"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available RAM</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_ram }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div>
          <label for="disk">Disk (GB)</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="disk"
                v-model="disk"
                class="input join-item validator w-full rounded-l-lg border p-2"
                :min="minDiskForCurrentTemplate"
                :max="userFreeResources?.free_disk"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip tooltip-top flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available Disk</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_disk }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="lifetime">Lifetime</label>
          <select class="select w-full rounded-lg border" v-model.number="lifetime">
            <option value="1" selected>1 Month</option>
            <option value="3">3 Months</option>
            <option value="6">6 Months</option>
            <option value="12">12 Months</option>
          </select>
        </div>
        <div>
          <label for="template">OS</label>
          <select class="select w-full rounded-lg border" v-model="template">
            <option v-for="t in readyTemplates" :key="t.name" :value="t.name">{{ t.name }}</option>
          </select>
        </div>
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
            <div class="flex justify-between gap-2 xl:gap-4">
              <div
                class="*:btn-sm col-span-2 grid max-w-48 min-w-24 flex-1 grid-cols-1 items-center gap-2 2xl:col-span-1 2xl:min-w-48 2xl:grid-cols-2"
              >
                <button
                  v-if="vm.status === 'stopped'"
                  @click="preStartVMWrapper(vm.id)"
                  :disabled="
                    isLoading(vm.id, 'start') ||
                    isVMExpired(vm.lifetime) ||
                    vm.group_role == 'member'
                  "
                  class="btn btn-success btn-outline col-span-2 min-w-24 rounded-lg"
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
                  class="btn btn-warning btn-outline min-w-20 rounded-lg"
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
                  class="btn btn-info btn-outline min-w-24 rounded-lg"
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
                  @click="preDeleteVM(vm.id)"
                  :disabled="vm.group_role == 'member' || isLoading(vm.id, 'delete')"
                  class="btn btn-error btn-outline col-span-2 min-w-24 rounded-lg"
                >
                  <span
                    v-if="isLoading(vm.id, 'delete')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:delete" class="text-lg" />
                  <span class="hidden lg:inline">Delete</span>
                </button>
              </div>
              <div>
                <RouterLink
                  v-if="vm.status !== 'pre-deleting' && vm.status !== 'deleting'"
                  :to="`/vm/${vm.id}`"
                  class="btn btn-primary btn-outline min-h-8 rounded-lg max-2xl:h-full"
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

    <VMStartChecksModal
      :model-value="showModal"
      :missing="modalMissing"
      :interfaces-href="`/vm/${startingVMId}/interfaces`"
      @confirm="confirmStart"
      @cancel="cancelStart"
    />

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete VM"
      positiveText="Delete VM"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteVM(vmToDelete!)"
      @negative="cancelDeleteVM(vmToDelete!)"
    >
      <p>Are you sure you want to delete this VM? This action cannot be undone.</p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
