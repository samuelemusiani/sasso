<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { VM } from '@/types'
import { api } from '@/lib/api'

const vms = ref()
const vmid = ref(0)

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
    .post('/vm')
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
    <div>
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
