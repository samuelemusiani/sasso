<script setup lang="ts">
const props = defineProps<{
  stat: {
    item: string
    icon: string
    total: number
    active: number
    max: number
    allocated: number
    color: string
  }
}>()

const colorClasses: Record<string, { bg: string; text: string; inner: string }> = {
  primary: { bg: 'bg-primary/20', text: '!text-primary', inner: '!text-primary-content' },
  secondary: { bg: 'bg-secondary/20', text: 'text-secondary', inner: 'text-secondary-content' },
  accent: { bg: 'bg-accent/20', text: 'text-accent', inner: 'text-accent-content' },
  info: { bg: 'bg-info/20', text: 'text-info', inner: 'text-info-content' },
  success: { bg: 'bg-success/20', text: 'text-success', inner: 'text-success-content' },
  warning: { bg: 'bg-warning/20', text: 'text-warning', inner: 'text-warning-content' },
  error: { bg: 'bg-error/20', text: 'text-error', inner: 'text-error-content' },
}
</script>
<template>
  <div
    class="flex flex-row justify-around items-center p-4 gap-1 rounded-xl shadow-md min-w-54 grow"
    :class="`bg-base-100 ${colorClasses[props.stat.color].text}`"
  >
    <div class="flex items-center flex-col gap-2">
      <IconVue class="text-4xl" :icon="props.stat.icon" />
      <span class="font-semibold">{{ props.stat.item }}</span>
    </div>
    <div class="flex flex-row gap-2 items-center">
      <div class="flex items-start">
        <span class="text-xl font-bold">{{ props.stat.allocated }} </span>
        <span v-if="props.stat.item === 'RAM' || props.stat.item === 'Disk'">GB</span>
      </div>
      <span class="text-3xl">/</span>
      <div class="flex items-start">
        <span class="text-xl font-bold">{{ props.stat.max }}</span>
        <span v-if="props.stat.item === 'RAM' || props.stat.item === 'Disk'">GB</span>
      </div>
    </div>
    <!-- TODO: change color base on props -->
    <div class="relative flex items-center justify-center">
      <div
        class="radial-progress absolute text-base-300"
        style="--value: 100; --size: 5.5rem; --thickness: 4px"
        role="progressbar"
      ></div>
      <div
        class="radial-progress absolute {{ colorClasses[props.stat.color].text }}"
        :style="`--value:${props.stat.total}; --size:6rem; --thickness:0.6rem;`"
        role="progressbar"
      ></div>
      <div
        class="absolute radial-progress absolute {{ colorClasses[props.stat.color].inner }}"
        :style="`--value:${props.stat.active}; --size:4rem; --thickness:0.5rem;`"
        role="progressbar"
      ></div>
      <span class="absolute text-sm font-bold">{{ props.stat.total }}% </span>
    </div>
  </div>
</template>
