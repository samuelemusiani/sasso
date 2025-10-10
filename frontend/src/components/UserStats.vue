<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  stats: Array<{
    item: string
    icon: string
    total: number
    active: number
    max: number
    allocated: number
    color: string
  }>
}>()

const colorClasses: Record<
  string,
  { bg: string; text: string; inner: string; hex: string; gradient: string }
> = {
  primary: {
    bg: 'bg-primary/20',
    text: '!text-primary',
    inner: '!text-primary-content',
    hex: '#365fab',
    gradient: 'from-primary/20 to-primary/5',
  },
  secondary: {
    bg: 'bg-secondary/20',
    text: 'text-secondary',
    inner: 'text-secondary-content',
    hex: '#f000b8',
    gradient: 'from-secondary/20 to-secondary/5',
  },
  orange: {
    bg: 'bg-orange-400/20',
    text: 'text-orange-400',
    inner: 'text-orange-400-content',
    hex: '#F7996E',
    gradient: 'from-orange-400/20 to-orange-400/5',
  },
  accent: {
    bg: 'bg-accent/20',
    text: 'text-accent',
    inner: 'text-accent-content',
    hex: '#37cdbe',
    gradient: 'from-accent/20 to-accent/5',
  },
  info: {
    bg: 'bg-info/20',
    text: 'text-info',
    inner: 'text-info-content',
    hex: '#3abff8',
    gradient: 'from-info/20 to-info/5',
  },
  success: {
    bg: 'bg-success/20',
    text: 'text-success',
    inner: 'text-success-content',
    hex: '#36d399',
    gradient: 'from-success/20 to-success/5',
  },
  warning: {
    bg: 'bg-warning/20',
    text: 'text-warning',
    inner: 'text-warning-content',
    hex: '#fbbd23',
    gradient: 'from-warning/20 to-warning/5',
  },
  error: {
    bg: 'bg-error/20',
    text: 'text-error',
    inner: 'text-error-content',
    hex: '#f87272',
    gradient: 'from-error/20 to-error/5',
  },
}
</script>

<template>
  <!-- Cards View -->
  <div class="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
    <div
      v-for="stat in stats"
      :key="stat.item"
      class="group bg-base-100 border-base-300/20 hover:border-base-300/40 relative overflow-hidden rounded-2xl border shadow-xl backdrop-blur-sm hover:shadow-2xl"
    >
      <!-- Card Content -->
      <div class="relative z-10 p-6">
        <!-- Header with Icon and Title -->
        <div class="mb-6 flex items-center justify-between">
          <div class="flex items-center gap-3">
            <IconVue :icon="stat.icon" class="text-3xl" :class="colorClasses[stat.color].text" />
            <h3 class="text-base-content text-lg font-bold">{{ stat.item }}</h3>
          </div>
        </div>

        <!-- Stats Numbers -->
        <div class="mb-6 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="text-3xl font-black" :class="colorClasses[stat.color].text">
              {{ stat.allocated }}
            </span>
            <div class="text-base-content/60">
              <span v-if="stat.item === 'RAM' || stat.item === 'Disk'" class="text-sm font-bold"
                >GB</span
              >
              <span v-else class="text-sm font-bold">{{ stat.item }}s</span>
              <div class="text-xs">allocated</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-base-content/80 text-xl font-bold">
              {{ stat.max }}
              <span v-if="stat.item === 'RAM' || stat.item === 'Disk'" class="text-sm">GB</span>
              <span v-else class="text-sm font-bold">{{ stat.item }}s</span>
            </div>
            <div class="text-base-content/60 text-xs">total</div>
          </div>
        </div>

        <!-- Stats Bars -->
        <div class="flex flex-col justify-end gap-3">
          <!-- Usage Bar -->
          <div :class="{ invisible: stat.active == 0 }" class="space-y-2">
            <div class="text-base-content/70 flex justify-between text-xs">
              <span>Active Usage</span>
              <span
                >{{ stat.active }} / {{ stat.allocated }}
                {{ stat.item === 'RAM' || stat.item === 'Disk' ? 'GB' : stat.item }}</span
              >
            </div>
            <div class="bg-base-300/30 h-2 w-full overflow-hidden rounded-full">
              <div
                class="h-full rounded-full shadow-sm transition-all duration-1000 ease-out"
                :style="`width: ${(stat.active / stat.max) * 100}%; background: linear-gradient(90deg, ${colorClasses[stat.color].hex}80, ${colorClasses[stat.color].hex})`"
              ></div>
            </div>
          </div>

          <!-- Total Bar -->
          <div class="space-y-2">
            <div class="text-base-content/70 flex justify-between text-xs">
              <span>Allocated</span>
              <span
                >{{ stat.allocated }} / {{ stat.max }}
                {{ stat.item === 'RAM' || stat.item === 'Disk' ? 'GB' : stat.item }}</span
              >
            </div>
            <div class="bg-base-300/30 h-2 w-full overflow-hidden rounded-full">
              <div
                class="h-full rounded-full shadow-sm transition-all duration-1000 ease-out"
                :style="`width: ${(stat.allocated / stat.max) * 100}%; background: linear-gradient(90deg, ${colorClasses[stat.color].hex}80, ${colorClasses[stat.color].hex})`"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
