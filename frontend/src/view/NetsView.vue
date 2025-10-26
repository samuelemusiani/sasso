<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, computed } from 'vue'
import type { Group, Net } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { getStatusClass } from '@/const'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()

const nets = ref<Net[]>([])
const formNetName = ref('')
const formNetVlanAware = ref(false)
const formNetGroupId = ref<number>()
const error = ref('')
const groups = ref<Group[]>([])

const modifying = ref(false)
const modifyingNetId = ref<number | null>(null)

function fetchNets() {
  api
    .get('/net')
    .then((res) => {
      // nets.value = res.data as Net[]
      res.data.sort((a: Net, b: Net) => a.id - b.id)
      nets.value = res.data as Net[]
    })
    .catch((err) => {
      error.value = 'Failed to fetch nets: ' + err.response.data
      console.error('Failed to fetch nets:', err)
    })
}

function fetchGroups() {
  api.get('/groups').then((res) => {
    groups.value = res.data as Group[]
  })
}

let intervalId: number | null = null

onMounted(() => {
  fetchNets()
  fetchGroups()
  intervalId = setInterval(() => {
    fetchNets()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
})

interface NetCreationBody {
  name: string
  vlanaware: boolean
  group_id?: number
}

function createOrModifyNet() {
  if (modifying.value) {
    modifyNet()
    return
  }
  createNet()
}

function createNet() {
  if (!formNetName.value) {
    error.value = 'Please provide a valid network name'
    return
  }

  const body: NetCreationBody = {
    name: formNetName.value,
    vlanaware: formNetVlanAware.value,
  }
  if (formNetGroupId.value) {
    body.group_id = formNetGroupId.value
  }

  api
    .post('/net', body)
    .then(() => {
      formNetName.value = ''
      formNetVlanAware.value = false
      fetchNets()
    })
    .catch((err) => {
      error.value = 'Failed to create net: ' + err.response.data
      console.error('Failed to create net:', err)
    })
}

function modifyNet() {
  if (!formNetName.value) {
    error.value = 'Please provide a valid network name'
    return
  }

  const body: NetCreationBody = {
    name: formNetName.value,
    vlanaware: formNetVlanAware.value,
  }

  api
    .put(`/net/${modifyingNetId.value}`, body)
    .then(() => {
      toggleModify(-1)
      fetchNets()
    })
    .catch((err) => {
      error.value = 'Failed to create net: ' + err.response.data
      console.error('Failed to create net:', err)
    })
}

function deleteNet(id: number) {
  if (!confirm('Are you sure you want to delete this network?')) {
    return
  }

  api
    .delete(`/net/${id}`)
    .then(() => {
      console.log(`Network ${id} deleted successfully`)
      fetchNets()
    })
    .catch((err) => {
      toastError(`Failed to delete network: ` + err.response.data)
      console.error(`Failed to delete network ${id}:`, err)
    })
}

function toggleModify(id: number) {
  if (id === -1) {
    modifying.value = false
    modifyingNetId.value = null
    formNetName.value = ''
    formNetVlanAware.value = false
    formNetGroupId.value = undefined
    return
  } else {
    modifying.value = true
    modifyingNetId.value = id
    const net = nets.value.find((n) => n.id === id)
    if (net) {
      formNetName.value = net.name
      formNetVlanAware.value = net.vlanaware
      formNetGroupId.value = net.group_id
    }
  }
}

const nonMemberGroups = computed(() => {
  return groups.value.filter((group) => group.role !== 'member')
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <h1 class="flex items-center gap-2 text-3xl font-bold">
      <IconVue class="text-primary" icon="ph:network"></IconVue>Networks
    </h1>

    <CreateNew
      :title="modifying ? 'Modify Network' : 'Network'"
      :hideCreate="modifying"
      :create="createOrModifyNet"
      :error="error"
      :open="modifying"
      @close="toggleModify(-1)"
    >
      <div class="flex flex-col gap-2">
        <label for="name">Network Name</label>
        <input
          required
          v-model="formNetName"
          type="text"
          placeholder="Network Name"
          class="border-primary rounded-lg border p-2"
        />

        <template v-if="!modifying">
          <label for="group">Group (Optional)</label>
          <select v-model="formNetGroupId" class="select select-bordered">
            <option :value="undefined">Me</option>
            <option v-for="group in nonMemberGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </option>
          </select>
        </template>

        <label class="flex cursor-pointer items-center gap-3">
          <input v-model="formNetVlanAware" type="checkbox" class="checkbox checkbox-primary" />
          <span class="label-text text-base-content">Enable VLAN support</span>
        </label>
      </div>
    </CreateNew>

    <table class="table w-full table-auto">
      <thead>
        <tr>
          <th class="">Name</th>
          <th class="">Owner</th>
          <th class="">Status</th>
          <th class="">VlanAware</th>
          <th class="">Subnet</th>
          <th class="">Gateway</th>
          <th class=""></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="net in nets"
          :key="net.id"
          class="hover"
          :class="net.group_name ? 'bg-base-200' : ''"
        >
          <td class="min-w-28 text-lg font-semibold">{{ net.name }}</td>
          <td class="">{{ net.group_name ? net.group_name : 'Me' }}</td>
          <td class="font-semibold capitalize" :class="getStatusClass(net.status)">
            {{ net.status }}
          </td>
          <td class="">{{ net.vlanaware }}</td>
          <td class="">{{ net.subnet }}</td>
          <td class="">{{ net.gateway }}</td>
          <td class="flex gap-8">
            <button
              v-if="net.status === 'ready'"
              @click="toggleModify(net.id)"
              class="btn btn-primary btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:edit" class="text-lg" />
              <p class="hidden md:inline">Edit</p>
            </button>
            <button
              v-if="net.status === 'ready' || net.status === 'unknown'"
              @click="deleteNet(net.id)"
              :disabled="net.group_role === 'member'"
              class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:delete" class="text-lg" />
              <p class="hidden md:inline">Delete</p>
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
