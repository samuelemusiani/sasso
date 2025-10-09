<script setup lang="ts">
import SidePanel from '@/components/SidePanel.vue'
import { RouterView } from 'vue-router'
</script>

<template>
  <SidePanel />
  <div class="flex-1 overflow-auto">
    <div class="bg-base-200 m-6 rounded-2xl p-4">
      <RouterView />
    </div>
  </div>
</template>
