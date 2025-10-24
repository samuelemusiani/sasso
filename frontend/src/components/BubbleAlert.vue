<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  type: 'info' | 'warning' | 'error' | 'success' | ''
  position?: 'top' | 'right' | 'bottom' | 'left'
  title?: string
}>()

const icon = computed(() => {
  switch (props.type) {
    case 'info':
      return 'material-symbols:info-outline'
    case 'warning':
      return 'mingcute:warning-line'
    case 'error':
      return 'mingcute:close-circle-line'
    case 'success':
      return 'mingcute:check-circle-line'
    default:
      return 'mingcute:question-line'
  }
})

const border = computed(() => {
  switch (props.type) {
    case 'info':
      return 'border-info'
    case 'warning':
      return 'border-warning'
    case 'error':
      return 'border-error'
    case 'success':
      return 'border-success'
    default:
      return 'border-base-content/20'
  }
})

const positionClass = computed(() => {
  switch (props.position) {
    case 'top':
      return 'tooltip-top'
    case 'right':
      return 'tooltip-right'
    case 'bottom':
      return 'tooltip-bottom'
    case 'left':
      return 'tooltip-left'
    default:
      return 'tooltip-bottom'
  }
})
</script>

<template>
  <div class="tooltip" :class="[`text-${props.type}`, positionClass]">
    <div
      class="card tooltip-content bg-base-100 rounded-box w-96 border"
      :class="[props.type === '' ? 'text-base-content/80 !font-medium' : '', border]"
    >
      <div tabindex="0" class="card-body">
        <p v-if="props.title" class="card-title font-bold">{{ props.title }}</p>
        <p class="text-left">
          <slot></slot>
        </p>
      </div>
    </div>
    <IconVue :icon="icon" class="cursor-pointer text-lg" />
  </div>
</template>
