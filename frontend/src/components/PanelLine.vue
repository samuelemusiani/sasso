<script setup lang="ts">
import { RouterLink } from 'vue-router'

const props = defineProps<{
  to: string
  icon: string
  label: string
  collapsed?: boolean
}>()
</script>
<template>
  <RouterLink
    class="btn btn-ghost hover:bg-primary/20 hover:text-base-content w-full justify-start gap-2 rounded-full transition-colors"
    :class="{ 'w-min !justify-center !rounded-2xl px-0': props.collapsed }"
    exact-active-class="backdrop-blur-sm bg-white/70 font-semibold text-base-100"
    :to="props.to"
  >
    <div class="btn btn-square btn-sm btn-primary m-1 rounded-xl p-0">
      <IconVue :icon="props.icon" class="text-lg" />
    </div>
    <span v-if="!props.collapsed">{{ props.label }}</span>
  </RouterLink>
</template>
