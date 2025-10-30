import { ref } from 'vue'

export interface Toast {
  id: number
  message: string
  type: 'info' | 'success' | 'warning' | 'error'
}

const toasts = ref<Toast[]>([])
let nextId = 0

export const useToastService = () => {
  const addToast = (message: string, type: Toast['type'] = 'info', duration = 3000) => {
    const id = nextId++
    toasts.value.push({ id, message, type })

    if (duration > 0) {
      setTimeout(() => {
        removeToast(id)
      }, duration)
    }

    return id
  }

  const removeToast = (id: number) => {
    const index = toasts.value.findIndex((t) => t.id === id)
    if (index > -1) {
      toasts.value.splice(index, 1)
    }
  }

  return {
    toasts,
    addToast,
    removeToast,
    success: (message: string, duration?: number) => addToast(message, 'success', duration),
    error: (message: string, duration?: number) => addToast(message, 'error', duration),
    warning: (message: string, duration?: number) => addToast(message, 'warning', duration),
    info: (message: string, duration?: number) => addToast(message, 'info', duration),
  }
}

// Export a singleton instance for use outside components
let toastService: ReturnType<typeof useToastService> | null = null

export const initToastService = () => {
  toastService = useToastService()
  return toastService
}

export const toast = {
  success: (message: string, duration?: number) => toastService?.success(message, duration),
  error: (message: string, duration?: number) => toastService?.error(message, duration),
  warning: (message: string, duration?: number) => toastService?.warning(message, duration),
  info: (message: string, duration?: number) => toastService?.info(message, duration),
}
