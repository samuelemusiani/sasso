<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { PortForward } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'

const pfs = ref<PortForward[]>([])
const port = ref(0)
const ip = ref('')

const publicIP = ref('')

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

function fetchPublicIP() {
  api
    .get('/port-forwards/public-ip')
    .then((res) => {
      publicIP.value = res.data.public_ip
    })
    .catch((err) => {
      console.error('Failed to fetch public IP:', err)
    })
}

onMounted(() => {
  fetchPortForwards()
  fetchPublicIP()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div>
      <p class="">
        The public IP is: <strong>{{ publicIP }}</strong>
      </p>
    </div>
    <CreateNew title="Port Forward" :create="requestPortForward">
      <div class="flex items-center gap-2">
        <label for="name">Destination Port</label>
        <input type="number" id="name" v-model="port" class="input w-48 rounded-lg border p-2" />
        <label for="key">Destination IP</label>
        <input type="text" id="key" v-model="ip" class="input w-96 rounded-lg border p-2" />
      </div>
    </CreateNew>

    <table class="table w-full table-auto">
      <thead>
        <tr>
          <th scope="col">Out Port</th>
          <th scope="col">Destination Port</th>
          <th scope="col">Destination IP</th>
          <th scope="col">Group</th>
          <th scope="col">Approved</th>
          <th scope="col" class="">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pf in pfs" :key="pf.id">
          <td class="whitespace-nowrap">{{ pf.out_port }}</td>
          <td class="whitespace-nowrap">{{ pf.dest_port }}</td>
          <td class="whitespace-nowrap">{{ pf.dest_ip }}</td>
          <td class="whitespace-nowrap">{{ pf.name || 'Me' }}</td>
          <td class="whitespace-nowrap">{{ pf.approved }}</td>
          <td class="whitespace-nowrap">
            <button
              @click="deletePortForward(pf.id)"
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
