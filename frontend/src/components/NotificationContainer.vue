<template>
  <Teleport to="body">
    <div class="notification-container">
      <Transition
        v-for="notification in notifications"
        :key="notification.id"
        name="notification"
        appear
      >
        <div
          :class="[
            'w-full shadow-2xl rounded-lg pointer-events-auto ring-1 ring-black ring-opacity-5 overflow-hidden backdrop-blur-sm transition-all duration-400',
            {
              'bg-error text-error-content': notification.type === 'error',
              'bg-success text-success-content': notification.type === 'success',
              'bg-warning text-warning-content': notification.type === 'warning',
              'bg-info text-info-content': notification.type === 'info',
              'opacity-0 transform translate-x-full scale-95': notification.isRemoving
            }
          ]"
        >
          <div class="p-4">
            <div class="flex items-start">
              <div class="flex-shrink-0">
                <Icon 
                  :icon="getIcon(notification.type)" 
                  class="text-xl"
                />
              </div>
              <div class="ml-3 w-0 flex-1">
                <p class="text-sm font-medium">
                  {{ notification.title }}
                </p>
                <p v-if="notification.message" class="mt-1 text-sm opacity-90">
                  {{ notification.message }}
                </p>
              </div>
              <div class="ml-4 flex-shrink-0 flex">
                <button
                  @click="removeNotification(notification.id)"
                  class="rounded-md inline-flex text-current hover:opacity-75 focus:outline-none"
                >
                  <Icon icon="material-symbols:close" class="text-lg" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const notifications = globalNotifications.notifications
const removeNotification = globalNotifications.removeNotification

function getIcon(type: string) {
  switch (type) {
    case 'error':
      return 'material-symbols:error'
    case 'success':
      return 'material-symbols:check-circle'
    case 'warning':
      return 'material-symbols:warning'
    case 'info':
      return 'material-symbols:info'
    default:
      return 'material-symbols:notifications'
  }
}
</script>

<style scoped>
/* Container posizionato correttamente */
.notification-container {
  position: fixed;
  top: 1rem;
  right: 1rem;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-width: 24rem;
  width: auto;
  min-width: 20rem;
  pointer-events: none;
}

.notification-container > * {
  pointer-events: auto;
}

.notification-enter-active,
.notification-leave-active {
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.notification-enter-from {
  opacity: 0;
  transform: translateX(100%) scale(0.95);
}

.notification-leave-to {
  opacity: 0;
  transform: translateX(100%) scale(0.95);
}

.notification-enter-to,
.notification-leave-from {
  opacity: 1;
  transform: translateX(0) scale(1);
}
</style>