<script setup lang="ts">
import { RouterLink } from 'vue-router'

const props = defineProps<{
  value: {
    to: string
    color: string
    icon: string
    title: string
  }
}>()
</script>

<template>
  <RouterLink
    class="card shadow-2xl border border-white/30 bg-gradient-to-br backdrop-blur-2xl backdrop-saturate-200 hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25),0_0_20px_rgba(255,255,255,0.1)] hover:scale-[1.02] hover:border-white/40 transition-all duration-300 ease-out overflow-hidden relative group before:absolute before:inset-0 before:bg-gradient-to-r before:from-transparent before:via-white/5 before:to-transparent before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300"
    :class="`bg-${props.value.color}`"
    :to="props.value.to"
  >
    <div class="card-body p-6">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div
            class="btn btn-square btn-lg rounded-xl p-0 bg-white/10 border-white/20 hover:bg-white/20"
          >
            <IconVue
              :icon="props.value.icon"
              class="text-2xl"
              :class="`text-${props.value.color}`"
            />
          </div>
          <div>
            <h3 class="font-bold text-xl text-base-content mb-1">{{ props.value.title }}</h3>
          </div>
        </div>
        <div
          class="btn btn-square btn-sm rounded-lg bg-white/10 border-white/20 group-hover:bg-white/20 transition-colors"
        >
          <IconVue icon="material-symbols:arrow-forward" class="text-lg" />
        </div>
      </div>
    </div>
  </RouterLink>
</template>
