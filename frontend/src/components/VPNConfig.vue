<script setup lang="ts">
import type { VPNConfig } from '@/types'
import { computed, ref } from 'vue'
import { copyToClipboard, downloadTextFile } from '@/lib/utils'
import { useToastService } from '@/composables/useToast'

const { success: toastSuccess } = useToastService()

const $props = defineProps<{
  vpnConfig: VPNConfig
}>()

const $emits = defineEmits<{
  (e: 'delete'): void
}>()

const copySuccess = ref(false)

async function copyConfig() {
  await copyToClipboard($props.vpnConfig.vpn_config)
  toastSuccess('VPN configuration copied to clipboard!')
}

function downloadConfig() {
  if (!$props.vpnConfig) {
    console.warn('No configuration available for download')
    return
  }

  downloadTextFile({
    text: $props.vpnConfig.vpn_config,
    filename: 'sasso-wireguard.conf',
  })
}

function deleteConfig() {
  alert('Are you sure you want to delete this configuration? This action cannot be undone.')
  $emits('delete')
}

const showKeys = ref(false)
const maskedConfig = computed(() => {
  if (!$props.vpnConfig) return ''

  return $props.vpnConfig.vpn_config.replace(
    /(PrivateKey\s*=\s*)([A-Za-z0-9+/=]+)/g,
    '$1' + '*'.repeat(35),
  )
})
</script>

<template>
  <div class="flex flex-col gap-4 p-2">
    <div class="flex justify-between">
      <div class="flex items-center gap-2">
        <button
          @click="copyConfig()"
          class="btn btn-outline btn-sm rounded-lg"
          :class="copySuccess ? 'btn-success' : 'btn-primary'"
        >
          <IconVue
            :icon="copySuccess ? 'material-symbols:check' : 'material-symbols:content-copy'"
            class="text-lg"
          />
          {{ copySuccess ? 'Copied!' : 'Copy' }}
        </button>

        <button @click="downloadConfig()" class="btn btn-primary btn-sm rounded-lg">
          <IconVue icon="material-symbols:download" class="text-lg" />
          Download .conf
        </button>
      </div>
      <button class="btn btn-error btn-sm rounded-lg" @click="deleteConfig">
        Delete Configuration
      </button>
    </div>
    <div class="bg-base-100/50 border-base-300/50 rounded-lg border p-4 whitespace-pre">
      <div class="mb-2 flex items-center justify-between">
        <p class="text-base-content/60 mb-2 text-xs font-semibold">sasso-wireguard.conf</p>
        <button class="badge badge-warning" @click="showKeys = !showKeys">
          <IconVue v-if="showKeys" icon="material-symbols:visibility-off" class="text-xs" />
          <IconVue v-else icon="material-symbols:visibility" class="text-xs" />
          {{ showKeys ? 'Hide' : 'Show' }} keys
        </button>
      </div>
      <p>{{ showKeys ? vpnConfig.vpn_config : maskedConfig }}</p>
    </div>
  </div>
</template>
