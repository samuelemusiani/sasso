<script setup lang="ts">
import { ref } from 'vue'
import { copyToClipboard } from '@/lib/utils'
import { useToastService } from '@/composables/useToast'

const { success: toastSuccess } = useToastService()

const props = defineProps<{ inline?: boolean }>()
const slotEl = ref<HTMLElement | null>(null)

async function copy() {
  const text = slotEl.value?.textContent?.trim() ?? ''
  if (!text) return

  await copyToClipboard(text)
  toastSuccess('Copied to clipboard!')
}
</script>

<template>
  <span v-if="props.inline" class="bg-base-100 rounded-lg p-1 font-mono">
    <slot />
  </span>

  <div v-else class="bg-base-100 flex items-center justify-between rounded-lg p-2">
    <div class="w-full overflow-x-scroll rounded-lg px-2 font-mono text-nowrap" ref="slotEl">
      <slot />
    </div>
    <div class="tooltip tooltip-left">
      <div class="tooltip-content rounded-lg border p-2" role="tooltip">
        <span>Copy to clipboard</span>
      </div>
      <button class="btn btn-ghost rounded-lg" @click="copy()">
        <IconVue icon="material-symbols:content-copy" class="text-lg" />
      </button>
    </div>
  </div>
</template>
