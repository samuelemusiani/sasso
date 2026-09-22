<script setup lang="ts">
import SidePanel from '@/components/SidePanel.vue'
import HelpPanel from '@/components/HelpPanel.vue'
import { RouterView } from 'vue-router'
</script>

<template>
  <SidePanel />
  <div class="flex-1 overflow-auto">
    <div class="bg-base-200 mx-6 my-4 rounded-2xl p-4">
      <RouterView />
    </div>
  </div>
  <HelpPanel />
</template>
