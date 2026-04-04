<script setup lang="ts">
import { useUiStore } from '@/stores/ui'
const ui = useUiStore()
</script>

<template>
  <button @click="ui.toggleHelp()" class="btn btn-info btn-outline rounded-lg">
    <IconVue icon="material-symbols:help-outline" class="text-lg" />
    HELP
  </button>
</template>
