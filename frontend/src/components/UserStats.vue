<script setup lang="ts">
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

const colors: Record<string, string> = {
  'text-primary': '#365fab',
  'text-secondary': '#27457c',
  'text-orange-400': '#f7996e',
  'text-accent': '#7ebdc3',
  'text-info': '#89d2dc',
  'text-success': '#a3f7b5',
  'text-warning': '#ffed65',
  'text-error': '#e87461'
}
</script>

<template>
  <!-- Cards View -->
  <div class="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
    <div
      v-for="stat in props.stats"
      :key="stat.item"
      class="group bg-base-100 border-base-300/20 hover:border-base-300/40 relative overflow-hidden rounded-2xl border shadow-xl backdrop-blur-sm hover:shadow-2xl"
    >
      <!-- Card Content -->
      <div class="relative z-10 p-6">
        <!-- Header with Icon and Title -->
        <div class="mb-6 flex items-center justify-between">
          <div class="flex items-center gap-3">
            <IconVue :icon="stat.icon" class="text-3xl" :class="`${stat.color}`" />
            <h3 class="text-base-content text-lg font-bold">{{ stat.item }}</h3>
          </div>
        </div>

        <!-- Stats Numbers -->
        <div class="mb-6 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="text-3xl font-black" :class="`${stat.color}`">
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
                :style="`width: ${(stat.active / stat.max) * 100}%; background: linear-gradient(90deg, ${colors[stat.color]}33, ${colors[stat.color]})`"
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
                :style="`width: ${(stat.allocated / stat.max) * 100}%; background: linear-gradient(90deg, ${colors[stat.color]}33, ${colors[stat.color]})`"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
