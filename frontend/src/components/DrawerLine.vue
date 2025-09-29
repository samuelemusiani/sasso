<script setup lang="ts">
import { RouterLink } from 'vue-router'

const props = defineProps<{
  to: string
  icon: string
  label: string
  collapsed: boolean
}>()
</script>
<template>
  <RouterLink
    class="rounded-full btn btn-ghost justify-start hover:bg-primary/20 transition-colors gap-2 w-full hover:text-base-content"
    :class="{ 'px-0 !justify-center !rounded-2xl w-min': props.collapsed }"
    active-class="backdrop-blur-sm bg-white/70 font-semibold text-base-100"
    :to="props.to"
  >
    <div class="btn btn-square btn-sm rounded-xl btn-primary p-0 m-1">
      <IconifyIcon :icon="props.icon" class="text-lg" />
    </div>
    <span v-if="!props.collapsed">{{ props.label }}</span>
  </RouterLink>
</template>
