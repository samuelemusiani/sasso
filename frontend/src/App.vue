<script setup lang="ts">
import { RouterView } from 'vue-router'
import router from '@/router/index.ts'
import SidePanel from '@/components/SidePanel.vue'

console.log('Current route:', router.currentRoute.value)
</script>

<template>
  <main
    class="flex w-full h-screen overflow-auto bg-[url(/sasso-pattern.png)] bg-base-300 bg-center bg-repeat bg-size-[3000px]"
  >
    <SidePanel v-if="router.currentRoute.value.path !== '/login'" />

    <div class="flex-1 overflow-auto">
      <div
        :class="`${router.currentRoute.value.path === '/login' ? '' : 'bg-base-200 p-4 rounded-2xl m-6'} `"
      >
        <RouterView />
      </div>
    </div>
  </main>
</template>

<!-- TODO: ip/24 converter -->
