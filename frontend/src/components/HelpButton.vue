<script setup lang="ts">
import { useUiStore } from '@/stores/ui'
import { getPageIcon } from '@/const'
const ui = useUiStore()
</script>

<template>
  <button @click="ui.toggleHelp()" class="btn btn-info btn-outline rounded-lg">
    <IconVue :icon="`${getPageIcon('help')}-outline`" class="text-lg" />
    HELP
  </button>
</template>
