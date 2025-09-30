<script setup lang="ts">
import { api } from '@/lib/api'
import { computed, onMounted, ref } from 'vue'

const vpnConfig = ref('')
const copySuccess = ref(false)

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

async function copyConfig() {
  try {
    await navigator.clipboard.writeText(vpnConfig.value)
    copySuccess.value = true
    setTimeout(() => {
      copySuccess.value = false
    }, 2000)
  } catch (error) {
    console.error('Error copying VPN config:', error)
  }
}

function downloadConfig() {
  if (!vpnConfig.value) {
    console.warn('No configuration available for download')
    return
  }

  const blob = new Blob([vpnConfig.value], { type: 'text/plain' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'sasso-wireguard.conf'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

const showKeys = ref(false)
const maskedConfig = computed(() => {
  if (!vpnConfig.value) return ''

  return vpnConfig.value.replace(/(PrivateKey\s*=\s*)([A-Za-z0-9+/=]+)/g, '$1' + '*'.repeat(35))
})
</script>

<template>
  <div class="p-2 flex flex-col gap-4">
    <div class="flex items-center gap-2">
      <button
        @click="copyConfig()"
        class="btn btn-outline btn-sm gap-2 rounded-lg"
        :class="copySuccess ? 'btn-success' : 'btn-primary'"
      >
        <IconVue
          :icon="copySuccess ? 'material-symbols:check' : 'material-symbols:content-copy'"
          class="text-lg"
        />
        {{ copySuccess ? 'Copied!' : 'Copy' }}
      </button>

      <button @click="downloadConfig()" class="btn btn-primary btn-sm gap-2 rounded-lg">
        <IconVue icon="material-symbols:download" class="text-lg" />
        Download .conf
      </button>
    </div>
    <div class="whitespace-pre bg-base-100/50 rounded-lg p-4 border border-base-300/50">
      <div class="flex items-center justify-between mb-2">
        <p class="text-xs text-base-content/60 font-semibold mb-2">sasso-wireguard.conf</p>
        <button class="badge badge-warning badge-sm" @click="showKeys = !showKeys">
          <IconVue v-if="!showKeys" icon="material-symbols:visibility-off" class="text-xs mr-1" />
          <IconVue v-else icon="material-symbols:visibility" class="text-xs mr-1" />
          {{ showKeys ? 'hide' : 'show' }} keys
        </button>
      </div>
      {{ showKeys ? vpnConfig : maskedConfig }}
    </div>
  </div>
</template>
