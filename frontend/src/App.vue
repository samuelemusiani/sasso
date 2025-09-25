<script setup lang="ts">
import { RouterView } from 'vue-router'
import Drawer from '@/components/Drawer.vue';
</script>

<template>
  <main
    class="flex w-full h-screen overflow-auto bg-[url(/pattern-sasso.png)] bg-base-300 bg-center bg-repeat bg-size-[3000px]">
    <Drawer />
    <div class="backdrop-blur-xl bg-base-200 p-4 rounded-2xl m-6 w-full">
      <RouterView />
    </div>
  </main>
</template>
