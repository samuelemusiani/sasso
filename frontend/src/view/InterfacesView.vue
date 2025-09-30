<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import type { Interface, Net } from '@/types'
import { api } from '@/lib/api'
import InterfaceForm from '@/components/vm/InterfaceForm.vue'

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

function fetchInterfaces() {
  api
    .get(`/vm/${vmid}/interface`)
    .then((res) => {
      interfaces.value = res.data as Interface[]
    })
    .catch((err) => {
      console.error('Failed to fetch interfaces:', err)
    })
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

onMounted(() => {
  fetchInterfaces()
  fetchNets()
})
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <h1 class="text-2xl">Manage Interfaces for VM {{ vmid }}</h1>

    <InterfaceForm :vmid="vmid" @interface-added="handleInterfaceAdded" @cancel="handleCancel" />
    <InterfaceForm
      v-if="editingInterface"
      :vmid="vmid"
      :interface="editingInterface"
      @interface-updated="handleInterfaceUpdated"
      @cancel="handleCancel"
    />
    <div class="alert alert-warning p-4" role="alert">
      <p class="font-bold">Warning</p>
      <p>
        Adding interfaces while the VM is running is possible. The VM will see the interface, but it
        will not be configured inside the VM. To have the interface configured, you will need to
        restart the VM.
      </p>
    </div>
    <div class="alert alert-info p-4" role="alert">
      <p class="font-bold">Information</p>
      <p>
        The VLAN tag is optional. If you don't know what to put here, leave it at zero. It could be
        used to separate different VMs at layer 2. Interfaces with the same VLAN tag can communicate
        with each other but not with interfaces with different VLAN tags. The gateway is on the
        untagged vlan (vlan 0). If you want to reach the internet with a VM, it needs to have at
        least one interface with vlan tag 0.
      </p>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-base-content">
        <thead class="bg-base-100">
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
              Network
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              VLAN Tag
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              IP Address
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Gateway
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Status
            </th>
            <th scope="col" class="relative px-6 py-3"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="iface in interfaces" :key="iface.id">
            <td class="px-6 py-4 whitespace-nowrap">{{ iface.id }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ netMap.get(iface.vnet_id) }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ iface.vlan_tag }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ iface.ip_add }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ iface.gateway }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ iface.status }}</td>
            <td
              class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium flex gap-2 justify-end"
            >
              <button @click="showEditForm(iface)" class="btn btn-primary p-2 rounded-lg">
                Edit
              </button>
              <button @click="deleteInterface(iface.id)" class="btn btn-error p-2 rounded-lg">
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
