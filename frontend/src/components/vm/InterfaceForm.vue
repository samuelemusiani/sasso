<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { api } from '@/lib/api'
import type { Interface, Net } from '@/types'
import CreateNew from '../CreateNew.vue'

const $props = defineProps<{
  vmid: number
  interface?: Interface
}>()

const $emit = defineEmits(['interfaceAdded', 'interfaceUpdated', 'cancel'])

const nets = ref<Net[]>([])
const editing = ref(!!$props.interface)

const form = ref({
  vnet_id: $props.interface?.vnet_id || 0,
  vlan_tag: $props.interface?.vlan_tag || 0,
  ip_add: $props.interface?.ip_add || '',
  gateway: $props.interface?.gateway || '',
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
    .post(`/vm/${$props.vmid}/interface`, form.value)
    .then(() => {
      $emit('interfaceAdded')
    })
    .catch((err) => {
      console.error('Failed to add interface:', err)
    })
}

function updateInterface() {
  if (!$props.interface) return
  api
    .put(`/vm/${$props.vmid}/interface/${$props.interface.id}`, form.value)
    .then(() => {
      $emit('interfaceUpdated')
    })
    .catch((err) => {
      console.error('Failed to update interface:', err)
    })
}

onMounted(() => {
  fetchNets()
})
</script>

<template>
  <CreateNew title="Interface" :create="handleSubmit">
    <h2 class="text-xl">{{ editing ? 'Edit' : 'Add' }} Interface</h2>
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <div>
        <label for="net" class="block text-sm font-medium">Network:</label>
        <select id="net" v-model="form.vnet_id" class="select rounded-lg w-full">
          <option v-for="net in nets" :key="net.id" :value="net.id">
            {{ net.name }}
          </option>
        </select>
      </div>
      <div>
        <div>Subnet: {{ currentSubnet }}</div>
        <div>Gateway: {{ currentGateway }}</div>
      </div>
      <div>
        <!-- FIXME: do this only if the net is vlanaware -->
        <label for="vlan_tag" class="block text-sm font-medium">VLAN Tag:</label>
        <input
          type="number"
          id="vlan_tag"
          v-model.number="form.vlan_tag"
          class="input rounded-lg w-full"
        />
      </div>
      <div>
        <!-- TODO: is needed to /24 -->
        <label for="ip_add" class="block text-sm font-medium">IP Address:</label>
        <input type="text" id="ip_add" v-model="form.ip_add" class="input rounded-lg w-full" />
      </div>
      <div>
        <!-- TODO: no /24  -->
        <label for="gateway" class="block text-sm font-medium">Gateway:</label>
        <input type="text" id="gateway" v-model="form.gateway" class="input rounded-lg w-full" />
      </div>
    </form>
  </CreateNew>
</template>
