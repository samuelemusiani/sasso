<script setup lang="ts">
import type { VM } from '@/types'

const $props = defineProps<{
  vm: VM
}>()
</script>

<template>
  <div>
    <div class="flex flex-col gap-2">
      <div>Cores: {{ $props.vm.cores }}</div>
      <div>RAM: {{ $props.vm.ram }}</div>
      <div>Disk: {{ $props.vm.disk }}</div>
    </div>
  </div>
</template>
