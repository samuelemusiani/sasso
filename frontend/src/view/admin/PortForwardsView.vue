<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { AdminPortForward } from '@/types'
import { api } from '@/lib/api'

const pfs = ref<AdminPortForward[]>([])
const port = ref(0)
const ip = ref('')

function fetchPortForwards() {
  api
    .get('/admin/port-forwards')
    .then((res) => {
      res.data.sort((a: AdminPortForward, b: AdminPortForward) => a.id - b.id)
      pfs.value = res.data as AdminPortForward[]
    })
    .catch((err) => {
      console.error('Failed to fetch Port Forwards:', err)
    })
}

function requestPortForward() {
  api
    .post('/admin/port-forwards', {
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

function approvePortForward(id: number, approve: boolean) {
  api
    .put(`/admin/port-forwards/${id}`, {
      approve: approve,
    })
    .then(() => {
      fetchPortForwards()
    })
    .catch((err) => {
      console.error('Failed to delete Port Forward:', err)
    })
}

onMounted(() => {
  fetchPortForwards()
})
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <div>This is the Port Forwards view for <b>sasso</b>!</div>
    <div class="flex gap-2 items-center">
      <label for="name">Destination Port:</label>
      <input type="number" id="name" v-model="port" class="border p-2 rounded-lg w-48" />
      <label for="key">Destinatio IP:</label>
      <input type="text" id="key" v-model="ip" class="border p-2 rounded-lg w-96" />
      <button class="bg-green-400 p-2 rounded-lg hover:bg-green-300" @click="requestPortForward()">
        Request Port Forward
      </button>
    </div>

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
                @click="approvePortForward(pf.id, !pf.approved)"
                class="bg-orange-400 p-2 rounded-lg hover:bg-orange-300"
              >
                {{ pf.approved ? 'Revoke' : 'Approve' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
