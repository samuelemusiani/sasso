import { ref, computed } from 'vue'
import type { AxiosInstance } from 'axios'
import type { UserResource } from '@/types'

type FreeResources = {
  free_cpu: number
  free_ram: number
  free_disk: number
  free_nets: number
}

export function useUserResources(api: AxiosInstance) {
  const userResources = ref<UserResource | null>(null)
  const userResourcesGB = computed<UserResource | null>(() => {
    if (!userResources.value) return null
    return {
      ...userResources.value,
      max_ram: userResources.value.max_ram / 1024,
      allocated_ram: userResources.value.allocated_ram / 1024,
      active_vms_ram: userResources.value.active_vms_ram / 1024,
      group_max_ram: userResources.value.group_max_ram / 1024,
    }
  })

  const userFreeResources = computed<FreeResources | null>(() => {
    if (!userResources.value) return null
    return {
      free_cpu: Math.max(
        0,
        userResources.value.max_cores -
          userResources.value.allocated_cores -
          userResources.value.group_max_cores,
      ),
      free_ram: Math.max(
        0,
        userResources.value.max_ram -
          userResources.value.allocated_ram -
          userResources.value.group_max_ram,
      ),
      free_disk: Math.max(
        0,
        userResources.value.max_disk -
          userResources.value.allocated_disk -
          userResources.value.group_max_disk,
      ),
      free_nets: Math.max(
        0,
        userResources.value.max_nets -
          userResources.value.allocated_nets -
          userResources.value.group_max_nets,
      ),
    }
  })

  const userFreeResourcesGB = computed<FreeResources | null>(() => {
    const free = userFreeResources.value
    if (!free) return null
    return {
      ...free,
      free_ram: free.free_ram / 1024,
    }
  })

  async function fetchUserResources() {
    try {
      const response = await api.get('/resources')
      userResources.value = response.data
    } catch (error) {
      console.error('Failed to fetch user resources:', error)
    }
  }

  return {
    userResources,
    fetchUserResources,
    userResourcesGB,
    userFreeResources,
    userFreeResourcesGB,
  }
}
