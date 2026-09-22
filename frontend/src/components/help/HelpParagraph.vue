<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()

// This is true only if the help page is fullscreen (we are under /help/*)
const isFullPage = computed(() => {
  return route.matched.some((r) => r.meta.fullscreen === true)
})
</script>

<template>
  <div
    class="flex flex-col"
    :class="{
      'gap-2': !isFullPage,
      'gap-3': isFullPage,
    }"
  >
    <h2
      class="text-info font-bold"
      :class="{
        'text-lg': !isFullPage,
        'text-2xl': isFullPage,
      }"
    >
      <slot name="title" />
    </h2>
    <slot />
  </div>
</template>
