<script setup lang="ts">

</script>

<template>
  <!-- <div class="bg-linear-to-r to-primary/50 from-base-100 h-full w-full">/ -->
  <div class=" h-full w-full">
    <div class="hidden">
      <div>Home view for <b>sasso</b>!</div>
    </div>
    <div v-if="whoami">
      {{ whoami }}
    </div>
    <Drawer></Drawer>
    <div class="flex gap-2"></div>
  </div>
</template>
