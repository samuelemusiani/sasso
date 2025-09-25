<script setup lang="ts">
interface Props {
  title: string
  percentage: number
  icon: string
  color?: string
}

const props = withDefaults(defineProps<Props>(), {
  color: 'primary'
})

// Calcola il circumference per l'animazione del cerchio
const radius = 40
const circumference = 2 * Math.PI * radius
const strokeDasharray = circumference
const strokeDashoffset = circumference - (props.percentage / 100) * circumference

// Colori dinamici basati sulla percentuale
const getColor = () => {
  if (props.percentage >= 80) return 'text-error'
  if (props.percentage >= 60) return 'text-warning'
  return 'text-success'
}
</script>

<template>
  <div class="card bg-base-100 shadow-xl border border-base-200 hover:shadow-2xl transition-all duration-300">
    <div class="card-body p-6">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-3 rounded-full bg-primary/10">
            <Icon :icon="icon" class="text-2xl text-primary" />
          </div>
          <div>
            <h3 class="font-bold text-lg">{{ title }}</h3>
            <p class="text-sm opacity-70">Sistema</p>
          </div>
        </div>
        
        <!-- Indicatore circolare di percentuale -->
        <div class="relative">
          <svg class="w-20 h-20 transform -rotate-90" viewBox="0 0 100 100">
            <!-- Cerchio di sfondo -->
            <circle
              cx="50"
              cy="50"
              :r="radius"
              stroke="currentColor"
              stroke-width="8"
              fill="none"
              class="text-base-300"
            />
            <!-- Cerchio di progresso -->
            <circle
              cx="50"
              cy="50"
              :r="radius"
              stroke="currentColor"
              stroke-width="8"
              fill="none"
              :stroke-dasharray="strokeDasharray"
              :stroke-dashoffset="strokeDashoffset"
              stroke-linecap="round"
              :class="getColor()"
              class="transition-all duration-1000 ease-out"
            />
          </svg>
          <!-- Percentuale al centro -->
          <div class="absolute inset-0 flex items-center justify-center">
            <span class="text-lg font-bold" :class="getColor()">
              {{ percentage }}%
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>