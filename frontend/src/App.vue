<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <main
    class="flex w-full h-screen overflow-auto bg-[url(/sasso-pattern.png)] bg-base-300 bg-center bg-repeat bg-size-[3000px]"
  >
    <RouterView />
  </main>
</template>

<!-- TODO: ip/24 converter -->
