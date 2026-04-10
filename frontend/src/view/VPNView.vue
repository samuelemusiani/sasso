<script setup lang="ts">
import { api } from '@/lib/api'
import type { VPNConfig } from '@/types'
import { onMounted, ref, watch, onBeforeUnmount } from 'vue'
import VPNConfigComponent from '@/components/VPNConfig.vue'
import { useToastService } from '@/composables/useToast'
import { useLoadingStore } from '@/stores/loading'
import { getPageIcon } from '@/const'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const vpnConfig = ref<VPNConfig[]>([])

const errorMessage = ref('')

function fetchVPNConfig() {
  loading.start('vpnConfig', null, 'fetch')
  api
    .get('/vpn/wireguard')
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
      toastError('Failed to fetch VPN configuration: ' + err.response)
    })
    .finally(() => {
      loading.stop('vpnConfig', null, 'fetch')
    })
}

function fetchVPNConfigWithoutLoading() {
  api
    .get('/vpn/wireguard')
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
      toastError('Failed to fetch VPN configuration: ' + err.response)
    })
}

let intervalId: number | null = null
const pendingConfigs = ref(0)

watch(pendingConfigs, (newValue, oldValue) => {
  if (newValue < oldValue) {
    fetchVPNConfigWithoutLoading()
  }

  if (newValue === 0 && intervalId) {
    console.log('No more pending configs, clearing interval')
    clearInterval(intervalId)
    intervalId = null
  }
})

function fetchCountPendingConfigs() {
  api
    .get('/vpn/wireguard?pending=true')
    .then((res) => {
      pendingConfigs.value = res.data.pending
      if (pendingConfigs.value > 0 && !intervalId) {
        intervalId = setInterval(() => {
          fetchCountPendingConfigs()
        }, 1000)
      }
    })
    .catch((err) => {
      console.error('Failed to fetch pending VPN configs:', err)
      toastError('Failed to fetch pending VPN configurations: ' + err.response)
    })
}

function newVPNConfig() {
  loading.start('vpnConfig', null, 'create')
  api
    .post('/vpn/wireguard')
    .then(() => {
      pendingConfigs.value += 1
      intervalId = setInterval(() => {
        fetchCountPendingConfigs()
      }, 1000)
    })
    .catch((err) => {
      console.error('Failed to create new VPN config:', err)
      if (err.response && err.response.data) {
        errorMessage.value = err.response.data
      }
      toastError('Failed to create new VPN configuration.')
    })
    .finally(() => {
      loading.stop('vpnConfig', null, 'create')
    })
}

function deleteVPN(id: number) {
  api
    .delete(`/vpn/wireguard/${id}`)
    .then(() => {
      fetchVPNConfigWithoutLoading()
    })
    .catch((err) => {
      console.error('Failed to delete VPN config:', err)
      toastError('Failed to delete VPN configuration.')
    })
}

onMounted(() => {
  fetchVPNConfig()
  fetchCountPendingConfigs()
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h2 class="card-title text-base-content flex items-center gap-3 text-3xl font-bold">
        <IconVue :icon="getPageIcon('vpn')" class="text-primary" />
        WireGuard VPN
      </h2>
      <HelpButton />
    </div>

    <div v-if="loading.is('vpnConfig', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <div v-else>
      <div v-for="config in vpnConfig" :key="config.id" class="my-4">
        <VPNConfigComponent :vpnConfig="config" @delete="deleteVPN(config.id)" />
      </div>
      <div v-for="n in pendingConfigs" :key="'pending-' + n" class="my-4">
        <VPNConfigComponent :skeleton="true" :vpnConfig="{} as VPNConfig" />
      </div>
      <div class="flex flex-col items-center gap-4">
        <button
          @click="newVPNConfig"
          class="btn btn-primary rounded-lg"
          :disabled="loading.is('vpnConfig', null, 'create')"
        >
          <span
            v-if="loading.is('vpnConfig', null, 'create')"
            class="loading loading-spinner loading-sm"
          />
          <IconVue v-else icon="material-symbols:add" class="text-lg" />
          Create New VPN Configuration
        </button>
        <p v-if="errorMessage" class="text-error mt-2">{{ errorMessage }}</p>
      </div>
    </div>
  </div>
</template>
