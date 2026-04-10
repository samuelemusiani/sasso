<script setup lang="ts">
import type { VPNConfig } from '@/types'
import { computed, ref } from 'vue'
import { copyToClipboard, downloadTextFile } from '@/lib/utils'
import { useToastService } from '@/composables/useToast'
import { useLoadingStore } from '@/stores/loading'
import ModalAlert from '@/components/ModalAlert.vue'

const { success: toastSuccess } = useToastService()
const loading = useLoadingStore()

const $props = defineProps<{
  vpnConfig: VPNConfig
  skeleton?: boolean
}>()

const $emits = defineEmits<{
  (e: 'delete'): void
}>()

const uniqueId = computed(() => ($props.vpnConfig ? $props.vpnConfig.id : Math.random()))

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

const showDeleteModal = ref(false)

function preDeleteConfig() {
  loading.start('vpnConfig', uniqueId.value, 'delete')
  showDeleteModal.value = true
}

function deleteConfig() {
  $emits('delete')
}

function cancelDeleteConfig() {
  loading.stop('vpnConfig', uniqueId.value, 'delete')
  showDeleteModal.value = false
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
  <div class="flex flex-col gap-4 p-2" :class="{ skeleton: skeleton }">
    <div class="flex justify-between">
      <div class="flex items-center gap-2">
        <button
          @click="copyConfig()"
          class="btn btn-outline btn-sm rounded-lg"
          :class="copySuccess ? 'btn-success' : 'btn-primary'"
          :disabled="skeleton"
        >
          <IconVue
            :icon="copySuccess ? 'material-symbols:check' : 'material-symbols:content-copy'"
            class="text-lg"
          />
          {{ copySuccess ? 'Copied!' : 'Copy' }}
        </button>

        <button
          @click="downloadConfig()"
          class="btn btn-primary btn-sm rounded-lg"
          :disabled="skeleton"
        >
          <IconVue icon="material-symbols:download" class="text-lg" />
          Download .conf
        </button>
      </div>
      <button
        class="btn btn-error btn-sm rounded-lg"
        @click="preDeleteConfig"
        :disabled="skeleton || loading.is('vpnConfig', uniqueId, 'delete')"
      >
        <span
          v-if="loading.is('vpnConfig', uniqueId, 'delete')"
          class="loading loading-spinner loading-xs"
        ></span>
        <IconVue v-else icon="material-symbols:delete" class="text-lg" />
        Delete
      </button>
    </div>
    <div class="bg-base-100/50 border-base-300/50 rounded-lg border p-4 whitespace-pre">
      <div class="mb-2 flex items-center justify-between">
        <p class="text-base-content/60 mb-2 text-xs font-semibold">sasso-wireguard.conf</p>
        <button
          class="btn btn-warning btn-sm rounded-lg"
          @click="showKeys = !showKeys"
          :disabled="skeleton"
        >
          <IconVue v-if="showKeys" icon="material-symbols:visibility-off" class="text-xs" />
          <IconVue v-else icon="material-symbols:visibility" class="text-xs" />
          {{ showKeys ? 'Hide' : 'Show' }} keys
        </button>
      </div>
      <p v-if="skeleton" class="grid h-32">
        <span class="loading loading-spinner loading-lg place-self-center"></span>
      </p>
      <p v-else>
        {{ showKeys ? vpnConfig.vpn_config : maskedConfig }}
      </p>
    </div>

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete Configuration"
      positiveText="Delete Configuration"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteConfig()"
      @negative="cancelDeleteConfig()"
    >
      <p>Are you sure you want to delete this configuration? This action cannot be undone.</p>
    </ModalAlert>
  </div>
</template>
