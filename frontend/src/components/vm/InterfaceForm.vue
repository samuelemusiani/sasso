<script setup lang="ts">
import { ref, onMounted, computed, watch, defineProps } from 'vue'
import { api } from '@/lib/api'
import type { Interface, Net, VM } from '@/types'
import CreateNew from '../CreateNew.vue'
import BubbleAlert from '../BubbleAlert.vue'

const $props = defineProps<{
  vm: VM
  interface?: Interface
}>()

const $emit = defineEmits(['interfaceAdded', 'interfaceUpdated', 'cancel'])

const nets = ref<Net[]>([])
const editing = computed(() => !!$props.interface)
const currentNet = computed(() => {
  return nets.value.find((n) => n.id === form.value.vnet_id)
})
const error = ref('')

const form = ref({
  vnet_id: $props.interface?.vnet_id || 0,
  vlan_tag: $props.interface?.vlan_tag || 0,
  ip_add: $props.interface?.ip_add || '',
  gateway: $props.interface?.gateway || '',
})

const filteredNets = computed(() => {
  const isGroupVM = $props.vm.group_id !== undefined
  if (!isGroupVM) {
    return nets.value.filter((net) => net.group_id === undefined)
  } else {
    return nets.value.filter(
      (net) => net.group_role !== 'member' && net.group_id === $props.vm.group_id,
    )
  }
})

const currentSubnet = computed(() => {
  const net = nets.value.find((n) => n.id === form.value.vnet_id)
  return net ? net.subnet : ''
})

const currentGateway = computed(() => {
  const net = nets.value.find((n) => n.id === form.value.vnet_id)
  return net ? net.gateway : ''
})

watch(
  () => form.value.vnet_id,
  (newVnetId) => {
    const net = nets.value.find((n) => n.id === newVnetId)
    if (net) {
      form.value.gateway = net.gateway
    } else {
      form.value.gateway = ''
    }
  },
)

watch(
  () => filteredNets.value,
  (newNets) => {
    if (newNets.length > 0 && !newNets.find((n) => n.id === form.value.vnet_id)) {
      form.value.vnet_id = newNets[0].id
    }
  },
  { immediate: true },
)

function fetchNets() {
  api
    .get('/net')
    .then((res) => {
      nets.value = res.data as Net[]
      if (!$props.interface && nets.value.length > 0) {
        form.value.vnet_id = nets.value[0].id
      }
    })
    .catch((err) => {
      console.error('Failed to fetch nets:', err)
      error.value = 'Failed to fetch networks: ' + err.response.data
    })
}

function handleSubmit() {
  if (editing.value) {
    updateInterface()
  } else {
    addInterface()
  }
}

function addInterface() {
  api
    .post(`/vm/${$props.vm.id}/interface`, form.value)
    .then(() => {
      $emit('interfaceAdded')
    })
    .catch((err) => {
      console.error('Failed to add interface:', err)
      error.value = 'Failed to add interface: ' + err.response.data
    })
}

function updateInterface() {
  if (!$props.interface) return
  api
    .put(`/vm/${$props.vm.id}/interface/${$props.interface.id}`, form.value)
    .then(() => {
      $emit('interfaceUpdated')
    })
    .catch((err) => {
      console.error('Failed to update interface:', err)
      error.value = 'Failed to update interface: ' + err.response.data
    })
}

onMounted(() => {
  fetchNets()
})
</script>

<template>
  <CreateNew
    :open="editing"
    :title="(editing ? 'Edit ' : '') + 'Interface'"
    :create="handleSubmit"
    :error="error"
    :hideCreate="editing"
    @close="$emit('cancel')"
  >
    <h2 class="text-xl">{{ editing ? 'Edit' : 'Add' }} Interface</h2>
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div>
        <label for="net" class="block text-sm font-medium">Network</label>
        <select id="net" v-model="form.vnet_id" class="select w-full rounded-lg">
          <option v-for="net in filteredNets" :key="net.id" :value="net.id">
            {{ net.name }}
          </option>
        </select>
      </div>

      <div>
        <div>Subnet: {{ currentSubnet }}</div>
        <div>Gateway: {{ currentGateway }}</div>
      </div>
      <div v-if="currentNet?.vlanaware">
        <div class="mb-1 flex items-center">
          <label for="vlan_tag" class="block text-sm font-medium">VLAN Tag</label>
          <BubbleAlert type="info" title="VLAN Tag"
            >The VLAN tag is optional. If you don't know what to put here, leave it at zero. It
            could be used to separate different VMs at layer 2. Interfaces with the same VLAN tag
            can communicate with each other but not with interfaces with different VLAN tags. The
            gateway is on the untagged vlan (vlan 0). If you want to reach the internet with a VM,
            it needs to have at least one interface with vlan tag 0.
          </BubbleAlert>
        </div>
        <input
          type="number"
          id="vlan_tag"
          v-model.number="form.vlan_tag"
          class="input w-full rounded-lg"
        />
      </div>
      <div>
        <!-- TODO: is needed to /24 -->
        <label for="ip_add" class="block text-sm font-medium">IP Address</label>
        <input type="text" id="ip_add" v-model="form.ip_add" class="input w-full rounded-lg" />
      </div>
      <div>
        <!-- TODO: no /24  -->
        <label for="gateway" class="block text-sm font-medium">Gateway</label>
        <input type="text" id="gateway" v-model="form.gateway" class="input w-full rounded-lg" />
      </div>
    </form>
  </CreateNew>
</template>
