<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { PortForward } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import ModalAlert from '@/components/ModalAlert.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const pfs = ref<PortForward[]>([])
const port = ref(0)
const ip = ref('')

const publicIP = ref('')
const error = ref('')

function fetchPortForwards() {
  loading.start('portForwards', null, 'fetch')
  api
    .get('/port-forwards')
    .then((res) => {
      pfs.value = res.data as PortForward[]
    })
    .catch((err) => {
      console.error('Failed to fetch Port Forwards:', err)
      toastError('Failed to fetch Port Forwards: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('portForwards', null, 'fetch')
    })
}

function requestPortForward() {
  return api
    .post('/port-forwards', {
      dest_port: port.value,
      dest_ip: ip.value,
    })
    .then(() => {
      fetchPortForwards()
      port.value = 0
      ip.value = ''
      return true
    })
    .catch((err) => {
      console.error('Failed to add port forward:', err)
      error.value = 'Failed to add port forward: ' + err.response.data
      return false
    })
}

const showDeleteModal = ref(false)
const portForwardToDelete = ref<number | null>(null)

function preDeletePortForward(id: number) {
  portForwardToDelete.value = id
  showDeleteModal.value = true
  loading.start('portForward', id, 'delete')
}

function deletePortForward(id: number) {
  api
    .delete(`/port-forwards/${id}`)
    .then(() => {
      // Small optimization
      pfs.value = pfs.value.filter((pf) => pf.id !== id)
      fetchPortForwards()
    })
    .catch((err) => {
      console.error('Failed to delete Port Forward:', err)
    })
    .finally(() => {
      loading.stop('portForward', id, 'delete')
    })
}

function cancelDeletePortForward(id: number) {
  portForwardToDelete.value = null
  showDeleteModal.value = false
  loading.stop('portForward', id, 'delete')
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
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" icon="material-symbols:router"></IconVue>Port Forwards
      </h1>
      <HelpButton />
    </div>
    <div>
      <p class="">
        <span> The public IP is: </span>
        <span v-if="!publicIP" class="loading loading-dots loading-xs"></span>
        <span v-else class="font-mono font-bold">
          {{ publicIP }}
        </span>
      </p>
    </div>
    <CreateNew
      title="Port Forward"
      :create="requestPortForward"
      :close-on-create="true"
      :error="error"
    >
      <div class="flex items-center gap-2">
        <label for="name">Destination Port</label>
        <input type="number" id="name" v-model="port" class="input w-48 rounded-lg border p-2" />
        <label for="key">Destination IP</label>
        <input type="text" id="key" v-model="ip" class="input w-96 rounded-lg border p-2" />
      </div>
    </CreateNew>

    <div v-if="loading.is('portForwards', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="table w-full table-auto">
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
              @click="preDeletePortForward(pf.id)"
              class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
              :disabled="loading.is('portForward', pf.id, 'delete')"
            >
              <span
                v-if="loading.is('portForward', pf.id, 'delete')"
                class="loading loading-spinner loading-xs"
              ></span>
              <IconVue v-else icon="material-symbols:delete" class="text-lg" />
              <p class="hidden md:inline">Delete</p>
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete Port Forward"
      positiveText="Delete Port Forward"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deletePortForward(portForwardToDelete!)"
      @negative="cancelDeletePortForward(portForwardToDelete!)"
    >
      <p>Are you sure you want to delete this Port Forward? This action cannot be undone.</p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
