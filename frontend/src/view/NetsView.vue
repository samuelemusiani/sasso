<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { Net } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
const nets = ref<Net[]>([])
const newNetName = ref('')
const newNetVlanAware = ref(false)
const error = ref('')

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

let intervalId: number | null = null

onMounted(() => {
  fetchNets()
  intervalId = setInterval(() => {
    fetchNets()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
})

function createNet() {
  if (!newNetName.value || !newNetVlanAware.value) {
    return
  }

  api
    .post('/net', { name: newNetName.value, vlanaware: newNetVlanAware.value })
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
  // if (!confirm('Are you sure you want to delete this network?')) {
  //   return
  // }

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

function getStatusClass(status: string) {
  switch (status) {
    case 'ready':
      return 'text-success'
    case 'error':
    case 'deleting':
    case 'pre-deleting':
    case 'unknown':
      return 'text-error'
    case 'creating':
    case 'pre-creating':
      return 'text-warning'
    case 'pending':
      return 'text-info'
    default:
      return 'text-info'
  }
}
// unknown, pending, ready, error, creating, deleting, pre-creating, pre-deleting
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <h1 class="text-3xl font-bold mb-2">My Networks</h1>

    <CreateNew title="Network" :create="createNet" :error="error">
      <div class="flex flex-col gap-2">
        <label for="name">Network Name</label>
        <input
          required
          v-model="newNetName"
          type="text"
          placeholder="Network Name"
          class="p-2 border border-primary rounded-lg"
        />

        <label class="cursor-pointer flex items-center gap-3">
          <input v-model="newNetVlanAware" type="checkbox" class="checkbox checkbox-primary" />
          <span class="label-text text-base-content">Abilita supporto VLAN</span>
        </label>
      </div>
    </CreateNew>

    <table class="table table-auto w-full">
      <thead>
        <tr>
          <th class="">Name</th>
          <th class="">Status</th>
          <th class="">VlanAware</th>
          <th class="">Subnet</th>
          <th class="">Gateway</th>
          <th class=""></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="net in nets" :key="net.id">
          <td class="">{{ net.name }}</td>
          <td class="capitalize font-semibold" :class="getStatusClass(net.status)">
            {{ net.status }}
          </td>
          <td class="">{{ net.vlanaware }}</td>
          <td class="">{{ net.subnet }}</td>
          <td class="">{{ net.gateway }}</td>
          <td class="">
            <button
              v-if="net.status === 'ready'"
              @click="deleteNet(net.id)"
              class="btn btn-error rounded-lg btn-sm md:btn-md btn-outline"
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
