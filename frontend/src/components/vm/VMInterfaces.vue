<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import type { Interface, Net, VM } from '@/types'
import { api } from '@/lib/api'
import InterfaceForm from '@/components/vm/InterfaceForm.vue'
import { getStatusClass } from '@/const'
import { useLoadingStore } from '@/stores/loading'

const $props = defineProps<{
  vm: VM
}>()

const route = useRoute()
const vmid = Number(route.params.vmid)

const interfaces = ref<Interface[]>([])
const nets = ref<Net[]>([])
const showAddForm = ref(false)
const editingInterface = ref<Interface | null>(null)

const netMap = computed(() => {
  const map = new Map<number, string>()
  for (const net of nets.value) {
    map.set(net.id, net.name)
  }
  return map
})

const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

async function fetchInterfacesWithLoading() {
  loading.start('vm', vmid, 'fetch_interfaces')
  await fetchInterfaces()
  loading.stop('vm', vmid, 'fetch_interfaces')
}
async function fetchInterfaces() {
  try {
    const res = await api.get(`/vm/${vmid}/interface`)
    interfaces.value = res.data as Interface[]
  } catch (err) {
    console.error('Failed to fetch interfaces:', err)
  }
}

function fetchNets() {
  api
    .get('/net')
    .then((res) => {
      nets.value = res.data as Net[]
    })
    .catch((err) => {
      console.error('Failed to fetch nets:', err)
    })
}

function deleteInterface(ifaceid: number) {
  if (confirm('Are you sure you want to delete this interface?')) {
    api
      .delete(`/vm/${vmid}/interface/${ifaceid}`)
      .then(() => {
        fetchInterfaces()
      })
      .catch((err) => {
        console.error('Failed to delete interface:', err)
      })
  }
}

function handleInterfaceAdded() {
  showAddForm.value = false
  fetchInterfaces()
}

function handleInterfaceUpdated() {
  editingInterface.value = null
  fetchInterfaces()
}

function handleCancel() {
  showAddForm.value = false
  editingInterface.value = null
}

function showEditForm(iface: Interface) {
  editingInterface.value = iface
  showAddForm.value = false
}

let intervalId: number | null = null

onMounted(() => {
  fetchInterfacesWithLoading()
  fetchNets()

  intervalId = setInterval(() => {
    fetchInterfaces()
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
    <InterfaceForm
      v-if="!editingInterface && $props.vm.group_role !== 'member'"
      :vm="$props.vm"
      @interface-added="handleInterfaceAdded"
      @cancel="handleCancel"
      :disabled="interfaces.length >= 32"
    />
    <InterfaceForm
      v-if="editingInterface && $props.vm.group_role !== 'member'"
      :vm="$props.vm"
      :interface="editingInterface"
      @interface-updated="handleInterfaceUpdated"
      @cancel="handleCancel"
    />

    <div v-if="interfaces.length >= 32" class="text-error">
      Max number of interfaces reached. Delete one inteface to add another one.
    </div>

    <div
      v-if="$props.vm.status == 'running'"
      class="alert alert-warning flex w-max flex-col p-4"
      role="alert"
    >
      <p class="font-bold">Adding interfaces to a running VM</p>
      <ul class="list-disc pl-5">
        <li>You can attach new interfaces while the VM is running.</li>
        <li>
          The VM will detect them, but <strong>they will not be configured automatically</strong>.
        </li>
        <li>To apply configuration, you need to restart the VM.</li>
      </ul>
    </div>

    <div v-if="isLoading(vm.id, 'fetch_interfaces')" class="grid h-70">
      <span class="loading loading-spinner place-self-center"></span>
    </div>

    <div v-else class="overflow-x-auto">
      <table class="table min-w-full divide-y">
        <thead>
          <tr>
            <th scope="col">ID</th>
            <th scope="col">Network</th>
            <th scope="col">VLAN Tag</th>
            <th scope="col">IP Address</th>
            <th scope="col">Gateway</th>
            <th scope="col">Status</th>
            <th scope="col" class="relative"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="iface in interfaces" :key="iface.id">
            <td class="">{{ iface.id }}</td>
            <td class="">{{ netMap.get(iface.vnet_id) }}</td>
            <td class="">{{ iface.vlan_tag }}</td>
            <td class="">{{ iface.ip_add }}</td>
            <td class="">{{ iface.gateway }}</td>
            <td class="font-semibold capitalize" :class="getStatusClass(iface.status)">
              {{ iface.status }}
            </td>
            <td
              v-if="vm && vm.group_role !== 'member'"
              class="flex justify-end gap-2 text-right text-sm font-medium"
            >
              <!-- FIXME: editing will show another"CreateNew" component filled -->
              <button @click="showEditForm(iface)" class="btn btn-primary rounded-lg p-2">
                Edit
              </button>
              <button @click="deleteInterface(iface.id)" class="btn btn-error rounded-lg p-2">
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
