<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  type: 'info' | 'warning' | 'error' | ''
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
    default:
      return 'mingcute:question-line'
  }
})
</script>

<template>
  <div class="dropdown dropdown-hover dropdown-center" :class="`text-${props.type}`">
    <div tabindex="0" role="button" class="btn btn-circle btn-ghost btn-xs">
      <IconVue :icon="icon" class="text-lg cursor-pointer" />
    </div>
    <div
      tabindex="0"
      class="card card-sm dropdown-content arrow bg-base-100 w-70 rounded-box z-1 shadow-sm border mt-2"
      :class="props.type === '' ? 'text-base-content/80 !font-medium' : ''"
    >
      <div tabindex="0" class="card-body">
        <p v-if="props.title" class="card-title font-bold">{{ props.title }}</p>
        <p>
          <slot></slot>
        </p>
      </div>
    </div>
  </div>
</template>
