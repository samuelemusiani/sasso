<script setup lang="ts">
import { api } from '@/lib/api'
import type { VPNConfig } from '@/types'
import { onMounted, ref } from 'vue'
import VPNConfigComponent from '@/components/VPNConfig.vue'
import { useToastService } from '@/composables/useToast'

const { error: toastError, success: toastSuccess } = useToastService()

const vpnConfig = ref<VPNConfig[]>([])

const message = ref('')

function fetchVPNConfig() {
  api
    .get('/vpn')
    .then((res) => {
      const tmp = res.data as VPNConfig[]
      tmp.map((config) => {
        // Mask sensitive information
        config.vpn_config = atob(config.vpn_config)
        return config
      })
      vpnConfig.value = tmp
      console.log('VPN Config response:', res.data)
      // Handle the VPN config data as needed
    })
    .catch((err) => {
      console.error('Failed to fetch VPN config:', err)
    })
}

function newVPNConfig() {
  api
    .post('/vpn')
    .then(() => {
      message.value =
        'New VPN configuration will be available shortly. Refresh the page after waiting a moment.'
    })
    .catch((err) => {
      console.error('Failed to create new VPN config:', err)
      toastError('Failed to create new VPN configuration.')
    })
}

function deleteVPN(id: number) {
  api
    .delete(`/vpn/${id}`)
    .then(() => {
      fetchVPNConfig()
      toastSuccess('VPN configuration deleted successfully.')
    })
    .catch((err) => {
      console.error('Failed to delete VPN config:', err)
      toastError('Failed to delete VPN configuration.')
    })
}

onMounted(() => {
  fetchVPNConfig()
})
</script>

<template>
  <div>
    <h2 class="card-title text-base-content flex items-center gap-3 text-3xl font-bold">
      <IconVue icon="material-symbols:settings" class="text-primary" />
      WireGuard's Configuration File
    </h2>

    <div v-for="config in vpnConfig" :key="config.id" class="my-4">
      <VPNConfigComponent :vpnConfig="config" @delete="deleteVPN(config.id)" />
    </div>
    <div class="flex justify-center">
      <button @click="newVPNConfig" class="btn btn-primary rounded-lg">
        Create New VPN Configuration
      </button>
      <p v-if="message" class="mt-2 text-green-600">{{ message }}</p>
    </div>
  </div>
</template>
