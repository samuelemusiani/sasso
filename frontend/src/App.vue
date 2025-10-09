<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <main
    class="bg-base-300 flex h-screen w-full overflow-auto bg-[url(/sasso-pattern.png)] bg-size-[3000px] bg-center bg-repeat"
  >
    <RouterView />
  </main>
</template>

<!-- TODO: ip/24 converter -->
