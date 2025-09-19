<script setup lang="ts">
import { api } from '@/lib/api'
import { onMounted, ref } from 'vue'

var vpnConfig = ref('')

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
    <RouterLink
      class="bg-gray-400 hover:bg-gray-300 p-2 rounded-lg min-w-32 block text-center mb-4 max-w-96"
      to="/"
    >
      Back to Home
    </RouterLink>
    <div class="font-bold text-lg">VPN configuration for <b>sasso</b>!</div>
    <p class="whitespace-pre bg-gray-50 mt-4">
      {{ vpnConfig }}
    </p>
  </div>
</template>
