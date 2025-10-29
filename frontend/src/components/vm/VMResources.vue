<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/lib/api'
import type { VM } from '@/types'
import CreateNew from '@/components/CreateNew.vue'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()

const $props = defineProps<{
  vm: VM
}>()

const cores = ref($props.vm.cores)
const ram = ref($props.vm.ram)
const disk = ref($props.vm.disk)

const $emit = defineEmits(['update-vm'])

function updateResources() {
  api
    .patch(`/vm/${$props.vm.id}/resources`, {
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
    })
    .then(() => {
      $emit('update-vm')
    })
    .catch((err) => {
      toastError('Failed to update resources: ' + err.response.data)
      console.error('Failed to update resources:', err)
    })
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <CreateNew title="Modify Resources" :hideCreate="true" :create="updateResources">
      <div class="flex flex-col gap-2">
        <label for="cores">Cores</label>
        <input
          required
          v-model="cores"
          type="number"
          class="input w-full"
          :class="{ 'input-error': cores < 1 }"
        />
        <label for="ram">RAM (MB)</label>
        <input
          required
          v-model="ram"
          type="number"
          class="input w-full"
          :class="{ 'input-error': ram < 512 }"
        />
        <label for="disk">Disk (GB)</label>
        <input
          required
          v-model="disk"
          type="number"
          class="input w-full"
          :class="{ 'input-error': disk < $props.vm.disk }"
        />
      </div>
    </CreateNew>

    <div class="flex flex-col gap-2">
      <div>Cores: {{ $props.vm.cores }}</div>
      <div>RAM: {{ $props.vm.ram }}</div>
      <div>Disk: {{ $props.vm.disk }}</div>
    </div>
  </div>
</template>
