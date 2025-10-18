<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { Group, Net } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { getStatusClass } from '@/const'

const nets = ref<Net[]>([])
const newNetName = ref('')
const newNetVlanAware = ref(false)
const newNetGroupId = ref<number>()
const error = ref('')
const groups = ref<Group[]>([])

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

function createNet() {
  if (!newNetName.value) {
    error.value = 'Please provide a valid network name'
    return
  }

  const body: NetCreationBody = {
    name: newNetName.value,
    vlanaware: newNetVlanAware.value,
  }
  if (newNetGroupId.value) {
    body.group_id = newNetGroupId.value
  }

  api
    .post('/net', body)
    .then(() => {
      newNetName.value = ''
      newNetVlanAware.value = false
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
      error.value = 'Failed to delete network: ' + err.response.data
      console.error(`Failed to delete network ${id}:`, err)
    })
}
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <h1 class="mb-2 text-3xl font-bold">Networks</h1>

    <CreateNew title="Network" :create="createNet" :error="error">
      <div class="flex flex-col gap-2">
        <label for="name">Network Name</label>
        <input
          required
          v-model="newNetName"
          type="text"
          placeholder="Network Name"
          class="border-primary rounded-lg border p-2"
        />

        <label for="group">Group (Optional)</label>
        <select v-model="newNetGroupId" class="select select-bordered">
          <option :value="undefined">Me</option>
          <option v-for="group in groups" :key="group.id" :value="group.id">
            {{ group.name }}
          </option>
        </select>

        <label class="flex cursor-pointer items-center gap-3">
          <input v-model="newNetVlanAware" type="checkbox" class="checkbox checkbox-primary" />
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
          <td class="">{{ net.name }}</td>
          <td class="">{{ net.group_name ? net.group_name : 'Me' }}</td>
          <td class="font-semibold capitalize" :class="getStatusClass(net.status)">
            {{ net.status }}
          </td>
          <td class="">{{ net.vlanaware }}</td>
          <td class="">{{ net.subnet }}</td>
          <td class="">{{ net.gateway }}</td>
          <td class="">
            <button
              v-if="net.status === 'ready'"
              @click="deleteNet(net.id)"
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
