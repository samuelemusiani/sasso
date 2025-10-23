<script setup lang="ts">
import { api } from '@/lib/api'
import type { VPNConfig } from '@/types'
import { onMounted, ref } from 'vue'
import VPNConfigComponent from '@/components/VPNConfig.vue'

const vpnConfig = ref<VPNConfig[]>([])

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

onMounted(() => {
  fetchVPNConfig()
})
</script>

<template>
  <div>
    <div v-for="config in vpnConfig" :key="config.id" class="mb-4">
      <VPNConfigComponent :vpnConfig="config" />
    </div>
  </div>
</template>
