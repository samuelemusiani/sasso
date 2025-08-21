<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { VM } from '@/types'
import { api } from '@/lib/api'

const vms = ref()
const vmid = ref(0)
const cores = ref(1)
const ram = ref(1024)
const disk = ref(2048)

function fetchVMs() {
  api
    .get('/vm')
    .then((res) => {
      vms.value = res.data as VM[]
    })
    .catch((err) => {
      console.error('Failed to fetch VMs:', err)
    })
}

setInterval(() => {
  fetchVMs()
}, 5000)

function createVM() {
  api
    .post('/vm', {
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
    })
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to create VM:', err)
    })
}

function deleteVM() {
  api
    .delete(`/vm/${vmid.value}`)
    .then(() => {
      fetchVMs()
    })
    .catch((err) => {
      console.error('Failed to delete VM:', err)
    })
}

onMounted(() => {
  fetchVMs()
})
</script>

<template>
  <div class="p-2 flex flex-col gap-2">
    <div>This is the VM view for <b>sasso</b>!</div>
    <div>
      {{ vms }}
    </div>
    <div class="flex gap-2 items-center">
      <label for="cores">Cores:</label>
      <input type="number" id="cores" v-model="cores" class="border p-2 rounded-lg w-24" />
      <label for="ram">RAM (MB):</label>
      <input type="number" id="ram" v-model="ram" class="border p-2 rounded-lg w-24" />
      <label for="disk">Disk (MB):</label>
      <input type="number" id="disk" v-model="disk" class="border p-2 rounded-lg w-24" />
      <button class="bg-green-400 p-2 rounded-lg hover:bg-green-300" @click="createVM()">
        Create VM
      </button>
    </div>
    <div class="flex gap-2 items-center">
      <input v-model="vmid" type="number" class="border p-2 rounded-lg w-96" />
      <button class="bg-red-400 p-2 rounded-lg hover:bg-red-300" @click="deleteVM()">
        Delete VM
      </button>
    </div>
  </div>
</template>
