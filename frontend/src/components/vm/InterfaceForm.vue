<script setup lang="ts">
import { ref, onMounted, computed, watch, defineProps } from 'vue'
import { api } from '@/lib/api'
import { IPAddress, CIDR } from '@/lib/ipaddr'
import type { Interface, Net, VM } from '@/types'
import CreateNew from '../CreateNew.vue'
import BubbleAlert from '../BubbleAlert.vue'

const $props = defineProps<{
  vm: VM
  interface?: Interface
}>()

const defaultVlanTagMessage = `
The VLAN tag is optional. If you don't know what to put here, leave it at zero.
It could be used to separate different VMs at layer 2. Interfaces with the same
VLAN tag can communicate with each other but not with interfaces with different
VLAN tags. The gateway is on the untagged vlan (vlan 0). If you want to reach
the internet with a VM, the interface with the gateway must have vlan tag 0.`

const $emit = defineEmits(['interfaceAdded', 'interfaceUpdated', 'cancel'])

const nets = ref<Net[]>([])
const editing = computed(() => !!$props.interface)
const currentNet = computed(() => {
  return nets.value.find((n) => n.id === form.value.vnet_id)
})
const error = ref('')

const ipIsUsed = ref(false)
const checkingIP = ref(false)

const form = ref<{
  vnet_id: number
  vlan_tag: number | string
  ip_add: string
  gateway: string
}>({
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

const ipValidationResult = computed<{
  status: 'success' | 'warning' | 'error'
  message: string
}>(() => {
  const ip = form.value.ip_add
  try {
    const currentSubnet = currentNet.value?.subnet
    const cidr = CIDR.parse(ip)
    const subnet = CIDR.parse(currentSubnet || '')

    if (ipIsUsed.value) {
      return {
        status: 'warning',
        message: 'The IP address is already in use in the selected network.',
      }
    }

    if (!subnet.contains(cidr.ip) || cidr.mask !== subnet.mask) {
      return {
        status: 'warning',
        message: `The IP address is correct and could be added, but it is outside of the
selected network's subnet (${currentSubnet}). You can still add it, but you have to know  what
you are doing.`,
      }
    }

    if (cidr.isNetworkAddr()) {
      return {
        status: 'warning',
        message: `The IP address is correct and could be added, but it is the network address.
You can still add it, but you have to know  what you are doing.`,
      }
    }

    if (cidr.isMaxHost()) {
      return {
        status: 'warning',
        message: `The IP address is correct and could be added, but it is the gateway address.
You can still add it, but you have to know  what you are doing.`,
      }
    }

    if (cidr.isBroadcastAddr()) {
      return {
        status: 'warning',
        message: `The IP address is correct and could be added, but it is the broadcast address.
You can still add it, but you have to know  what you are doing.`,
      }
    }

    return {
      status: 'success',
      message: 'The IP address is valid and in the correct subnet.',
    }
  } catch {
    return {
      status: 'error',
      message: `The ip address must be in CIDR notation. Something like 192.168.0.1/20`,
    }
  }
})

const ipInputClasses = computed(() => {
  const status = ipValidationResult.value.status
  return {
    'input-success': status === 'success',
    'input-warning': status === 'warning',
    'input-error': status === 'error',
  }
})
const ipErrorMessage = computed(() => ipValidationResult.value.message)

const gatewayValidationResult = computed<{
  status: 'success' | 'error'
  message: string
}>(() => {
  const gateway = form.value.gateway
  if (gateway === '') {
    return {
      status: 'success',
      message:
        'The gateway is optional. Leave it empty to not set a default gateway for this interface.',
    }
  }

  try {
    const gtw = IPAddress.parse(gateway)
    let cidr: CIDR
    try {
      cidr = CIDR.parse(form.value.ip_add)
      if (!cidr.contains(gtw)) {
        return {
          status: 'error',
          message: `The gateway format is correct, but it is not in the same subnet as the IP address`,
        }
      }
    } catch {}
    return {
      status: 'success',
      message: 'The gateway is valid.',
    }
  } catch {
    return {
      status: 'error',
      message: `The gateway must be a valid IP address. It MUST NOT be in CIDR notation.`,
    }
  }
})

const gatewayInputClasses = computed(() => {
  const status = gatewayValidationResult.value.status
  return {
    'input-success': status === 'success',
    'input-error': status === 'error',
  }
})
const gatewayErrorMessage = computed(() => gatewayValidationResult.value.message)

const vlanTagValidationResult = computed(() => {
  const vlanTag = form.value.vlan_tag
  if (vlanTag === '' || typeof vlanTag !== 'number' || vlanTag < 0 || vlanTag > 4095) {
    return 'error'
  }
  return 'info'
})

const vlanTagClasses = computed(() => {
  const status = vlanTagValidationResult.value
  return {
    'input-info': status === 'info',
    'input-error': status === 'error',
  }
})

const vlanTagMessage = computed(() => {
  if (vlanTagValidationResult.value === 'error') {
    return 'The VLAN tag must be a number between 0 and 4095.'
  } else {
    return defaultVlanTagMessage
  }
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
      form.value.vnet_id = newNets[0]?.id || 0
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
        form.value.vnet_id = nets.value[0]?.id || 0
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
      form.value = {
        vnet_id: filteredNets.value[0]?.id || 0,
        vlan_tag: 0,
        ip_add: '',
        gateway: filteredNets.value[0]?.gateway || '',
      }
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

async function isIPUsed(ip: string, vnet_id: number, vlan_tag: number): Promise<boolean> {
  const res = await api.post('/ip-check', { ip, vnet_id, vlan_tag })
  return res.data.in_use as boolean
}

watch(
  [() => form.value.ip_add, () => form.value.vnet_id, () => form.value.vlan_tag],
  async ([newIp]) => {
    if (
      ipValidationResult.value.status === 'success' ||
      ipValidationResult.value.status === 'warning'
    ) {
      checkingIP.value = true
      const used = await isIPUsed(newIp, form.value.vnet_id, Number(form.value.vlan_tag))
      checkingIP.value = false
      if (used) {
        ipIsUsed.value = true
      } else {
        ipIsUsed.value = false
      }
    } else {
      ipIsUsed.value = false
      checkingIP.value = false
    }
  },
)

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

      <div class="grid w-70 grid-cols-2">
        <div>Subnet</div>
        <div>{{ currentNet?.subnet }}</div>
        <div>Gateway</div>
        <div>{{ currentNet?.gateway }}</div>
        <div>Broadcast</div>
        <div>{{ currentNet?.broadcast }}</div>
      </div>
      <div v-if="currentNet?.vlanaware">
        <label for="vlan_tag" class="block text-sm font-medium">VLAN Tag</label>
        <label class="input w-full rounded-lg" :class="vlanTagClasses">
          <BubbleAlert :type="vlanTagValidationResult" title="VLAN Tag" position="right"
            >{{ vlanTagMessage }}
          </BubbleAlert>
          <input type="number" id="vlan_tag" v-model.number="form.vlan_tag" />
        </label>
      </div>
      <div>
        <label for="ip_add" class="block text-sm font-medium">IP Address</label>
        <label class="input w-full rounded-lg" :class="ipInputClasses">
          <span v-show="checkingIP" class="loading loading-spinner loading-md text-success"></span>
          <BubbleAlert
            v-show="!checkingIP"
            :type="ipValidationResult.status"
            title="IP Address"
            position="right"
          >
            {{ ipErrorMessage }}
          </BubbleAlert>
          <input type="text" id="ip_add" v-model="form.ip_add" />
        </label>
      </div>
      <div>
        <label for="gateway" class="block text-sm font-medium">Gateway</label>
        <label class="input w-full rounded-lg" :class="gatewayInputClasses">
          <BubbleAlert :type="gatewayValidationResult.status" title="Gateway" position="right">
            {{ gatewayErrorMessage }}
          </BubbleAlert>
          <input type="text" id="gateway" v-model="form.gateway" />
        </label>
      </div>
    </form>
  </CreateNew>
</template>
