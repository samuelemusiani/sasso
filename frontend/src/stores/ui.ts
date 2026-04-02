import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  const helpOpen = ref(false)
  const openHelp = () => (helpOpen.value = true)
  const closeHelp = () => (helpOpen.value = false)
  const toggleHelp = () => (helpOpen.value = !helpOpen.value)

  return { helpOpen, openHelp, closeHelp, toggleHelp }
})
