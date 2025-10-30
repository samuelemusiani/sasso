<script setup lang="ts">
import { useToastService } from '@/composables/useToast'

const { toasts, removeToast } = useToastService()

const getAlertClass = (type: string) => {
  const classes = {
    success: 'alert-success',
    error: 'alert-error',
    warning: 'alert-warning',
    info: 'alert-info',
  }
  return classes[type as keyof typeof classes] || 'alert-info'
}
</script>

<template>
  <div class="toast toast-top toast-end z-50">
    <TransitionGroup name="toast">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="['alert', getAlertClass(toast.type)]"
        class=""
      >
        <span>{{ toast.message }}</span>
        <button class="btn btn-sm btn-ghost" @click="removeToast(toast.id)">
          <IconVue icon="material-symbols:close" class="text-lg" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}
</style>
