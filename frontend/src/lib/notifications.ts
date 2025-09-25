import { ref } from 'vue'

export interface Notification {
  id: string
  type: 'error' | 'success' | 'warning' | 'info'
  title: string
  message?: string
  duration?: number
  isRemoving?: boolean
}

// Global state
const notificationList = ref<Notification[]>([])
const activeTimers = new Map<string, number>()

let notificationId = 0

function generateId(): string {
  return `notification-${++notificationId}-${Date.now()}`
}

// Global functions
function addNotification(notification: Omit<Notification, 'id'>) {
  const id = generateId()
  const newNotification: Notification = {
    id,
    duration: notification.duration ?? 5000, // 5 secondi di default  
    ...notification
  }
  
  notificationList.value.push(newNotification)
  
  // Auto-remove dopo la durata specificata
  const duration = newNotification.duration || 5000
  
  const timerId = window.setTimeout(() => {
    removeNotification(id)
  }, duration)
  
  activeTimers.set(id, timerId)
  
  return id
}

function removeNotification(id: string) {
  // Cancella il timer se esiste
  const timerId = activeTimers.get(id)
  if (timerId) {
    window.clearTimeout(timerId)
    activeTimers.delete(id)
  }
  
  const notification = notificationList.value.find((n: Notification) => n.id === id)
  if (notification) {
    // Imposta lo stato di rimozione per attivare la transizione
    notification.isRemoving = true
    
    // Rimuovi definitivamente dopo la durata della transizione
    setTimeout(() => {
      const index = notificationList.value.findIndex((n: Notification) => n.id === id)
      if (index > -1) {
        notificationList.value.splice(index, 1)
      }
    }, 400) // 400ms corrisponde alla durata della transizione CSS
  }
}

export function useNotifications() {
  return {
    notifications: notificationList,
    addNotification,
    removeNotification,
    clearAllNotifications: () => { notificationList.value = [] },
    showError: (title: string, message?: string, duration?: number) => 
      addNotification({ type: 'error', title, message, duration }),
    showSuccess: (title: string, message?: string, duration?: number) => 
      addNotification({ type: 'success', title, message, duration }),
    showWarning: (title: string, message?: string, duration?: number) => 
      addNotification({ type: 'warning', title, message, duration }),
    showInfo: (title: string, message?: string, duration?: number) => 
      addNotification({ type: 'info', title, message, duration })
  }
}

// Export direct global functions
export const globalNotifications = {
  showSuccess: (title: string, message?: string, duration?: number) =>
    addNotification({ type: 'success', title, message, duration }),
  showError: (title: string, message?: string, duration?: number) =>
    addNotification({ type: 'error', title, message, duration }),
  showWarning: (title: string, message?: string, duration?: number) =>
    addNotification({ type: 'warning', title, message, duration }),
  showInfo: (title: string, message?: string, duration?: number) =>
    addNotification({ type: 'info', title, message, duration }),
  removeNotification,
  notifications: notificationList
}