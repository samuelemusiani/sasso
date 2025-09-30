<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { PortForward } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'

const pfs = ref<PortForward[]>([])
const port = ref(0)
const ip = ref('')

function fetchPortForwards() {
  api
    .get('/port-forwards')
    .then((res) => {
      pfs.value = res.data as PortForward[]
    })
    .catch((err) => {
      console.error('Failed to fetch Port Forwards:', err)
    })
}

function requestPortForward() {
  api
    .post('/port-forwards', {
      dest_port: port.value,
      dest_ip: ip.value,
    })
    .then(() => {
      fetchPortForwards()
      port.value = 0
      ip.value = ''
    })
    .catch((err) => {
      console.error('Failed to add port forward:', err)
    })
}

function deletePortForward(id: number) {
  if (confirm('Are you sure you want to delete this port forward?')) {
    api
      .delete(`/port-forwards/${id}`)
      .then(() => {
        fetchPortForwards()
      })
      .catch((err) => {
        console.error('Failed to delete Port Forward:', err)
      })
  }
}

onMounted(() => {
  fetchPortForwards()
})
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <CreateNew title="Port Forward" :create="requestPortForward">
      <div class="flex gap-2 items-center">
        <label for="name">Destination Port</label>
        <input type="number" id="name" v-model="port" class="input border p-2 rounded-lg w-48" />
        <label for="key">Destination IP</label>
        <input type="text" id="key" v-model="ip" class="input border p-2 rounded-lg w-96" />
      </div>
    </CreateNew>

    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
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
              Out Port
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Destination Port
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Destination IP
            </th>
            <th
              scope="col"
              class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              Approved
            </th>
            <th scope="col" class="relative px-6 py-3">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="pf in pfs" :key="pf.id">
            <td class="px-6 py-4 whitespace-nowrap">{{ pf.id }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ pf.out_port }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ pf.dest_port }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ pf.dest_ip }}</td>
            <td class="px-6 py-4 whitespace-nowrap">{{ pf.approved }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
              <button
                @click="deletePortForward(pf.id)"
                class="bg-red-400 p-2 rounded-lg hover:bg-red-300"
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
