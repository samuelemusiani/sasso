import { ref, computed } from 'vue'
import type { AxiosInstance } from 'axios'
import type { GroupResources, FreeResources } from '@/types'

export function useGroupResources(api: AxiosInstance) {
  const groupResources = ref<GroupResources | null>(null)
  const groupResourcesGB = computed<GroupResources | null>(() => {
    if (!groupResources.value) return null
    return {
      ...groupResources.value,
      max_ram: groupResources.value.max_ram / 1024,
      allocated_ram: groupResources.value.allocated_ram / 1024,
      active_vms_ram: groupResources.value.active_vms_ram / 1024,
    }
  })

  const groupFreeResources = computed<FreeResources | null>(() => {
    if (!groupResources.value) return null
    return {
      free_cpu: Math.max(0, groupResources.value.max_cores - groupResources.value.allocated_cores),
      free_ram: Math.max(0, groupResources.value.max_ram - groupResources.value.allocated_ram),
      free_disk: Math.max(0, groupResources.value.max_disk - groupResources.value.allocated_disk),
      free_nets: Math.max(0, groupResources.value.max_nets - groupResources.value.allocated_nets),
    }
  })

  const groupFreeResourcesGB = computed<FreeResources | null>(() => {
    const free = groupFreeResources.value
    if (!free) return null
    return {
      ...free,
      free_ram: free.free_ram / 1024,
    }
  })

  async function fetchGroupResources(id: number) {
    try {
      const response = await api.get(`/groups/${id}/resources`)
      groupResources.value = response.data as GroupResources
    } catch (error) {
      console.error('Failed to fetch group resources:', error)
    }
  }

  return {
    groupResources: groupResources,
    fetchGroupResources: fetchGroupResources,
    groupResourcesGB: groupResourcesGB,
    groupFreeResources,
    groupFreeResourcesGB,
  }
}
