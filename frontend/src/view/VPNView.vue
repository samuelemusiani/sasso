<script setup lang="ts">
import { api } from '@/lib/api'
import { onMounted, ref } from 'vue'

const vpnConfig = ref('')

function fetchVPNConfig() {
  api
    .get('/vpn')
    .then((res) => {
      vpnConfig.value = res.data as string
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
  <div class="p-2">
    <!-- TODO: add copy and download btn -->
    <p class="whitespace-pre bg-base-100 p-3 rounded-lg mt-4">
      {{ vpnConfig }}
    </p>
  </div>
</template>
