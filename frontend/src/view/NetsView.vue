<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Net } from '@/types'
import { api } from '@/lib/api'
const nets = ref<Net[]>([])
const newNetName = ref('')

function fetchNets() {
  api
    .get('/net')
    .then((res) => {
      // nets.value = res.data as Net[]
      res.data.sort((a: Net, b: Net) => a.id - b.id)
      nets.value = res.data as Net[]
    })
    .catch((err) => {
      console.error('Failed to fetch nets:', err)
    })
}

setInterval(() => {
  fetchNets()
}, 5000)

function createNet() {
  if (!newNetName.value) {
    return
  }

  api
    .post('/net', { name: newNetName.value })
    .then(() => {
      newNetName.value = ''
      fetchNets()
    })
    .catch((err) => {
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
      console.error(`Failed to delete network ${id}:`, err)
    })
}

function addtmp() {
  console.log('Adding temporary networks...')
  for (let i = 0; i < 10; i++) {
    newNetName.value = `net-${Math.random().toString(36).substring(2, 8)}`
    createNet()
  }
}

function deltmp() {
  console.log('Deleting temporary networks...')
  nets.value.forEach((net) => {
    deleteNet(net.id)
  })
}

onMounted(() => {
  fetchNets()
})
</script>

<template>
  <div class="p-2">
    <h1 class="text-2xl">My Networks</h1>
    <div class="my-4">
      <h2 class="text-xl">Create New Network</h2>
      <div class="flex gap-2">
        <input
          v-model="newNetName"
          type="text"
          placeholder="Network Name"
          class="p-2 border border-primary rounded-lg"
        />
        <button @click="createNet" class="btn btn-info rounded-lg">Create</button>
      </div>
    </div>

    <button class="bg-purple-400 p-2 rounded-lg hover:bg-purple-300 w-64" @click="addtmp">
      Add n interfaces
    </button>
    <button class="bg-red-400 p-2 rounded-lg hover:bg-red-300 w-64" @click="deltmp">
      Delete n interfaces
    </button>

    <div>
      <h2 class="text-xl">Existing Networks</h2>
      <table class="table-auto w-full">
        <thead>
          <tr>
            <th class="px-4 py-2">ID</th>
            <th class="px-4 py-2">Name</th>
            <th class="px-4 py-2">VlanAware</th>
            <th class="px-4 py-2">Status</th>
            <th class="px-4 py-2">Subnet</th>
            <th class="px-4 py-2">Gateway</th>
            <th class="px-4 py-2"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="net in nets" :key="net.id">
            <td class="border px-4 py-2">{{ net.id }}</td>
            <td class="border px-4 py-2">{{ net.name }}</td>
            <td class="border px-4 py-2">{{ net.vlanaware }}</td>
            <td class="border px-4 py-2">{{ net.status }}</td>
            <td class="border px-4 py-2">{{ net.subnet }}</td>
            <td class="border px-4 py-2">{{ net.gateway }}</td>
            <td class="border px-4 py-2">
              <button
                v-if="net.status === 'ready'"
                @click="deleteNet(net.id)"
                class="text-red-400 hover:blue-red-300 p-2 hover:underline"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
