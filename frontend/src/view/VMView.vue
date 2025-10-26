<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { computed, onMounted, ref, onBeforeUnmount } from 'vue'
import { api } from '@/lib/api'
import type { VM } from '@/types'
import { useLoadingStore } from '@/stores/loading'

const router = useRouter()
const route = useRoute()

const vmid = computed(() => {
  const id = route.params.vmid
  if (Array.isArray(id)) return Number(id[0])
  return id ? Number(id) : 0
})

const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

const tabs = [
  { id: 'info', label: 'Info', path: '', icon: 'ph:info' },
  { id: 'resources', label: 'Resources', path: 'resources', icon: 'ph:cpu' },
  { id: 'interfaces', label: 'Interfaces', path: 'interfaces', icon: 'ph:path' },
  { id: 'backups', label: 'Backups', path: 'backups', icon: 'material-symbols:backup-outline' },
]

const activeTab = computed(() => {
  const path = route.path
  for (const tab of tabs) {
    if (tab.path === '') {
      // Check if we're at the base path /vm/:vmid
      if (path === `/vm/${vmid.value}`) return tab.id
    } else {
      if (path.endsWith(`/${tab.path}`)) return tab.id
    }
  }
  return 'info'
})

function shouldDisableTab(tabId: string): boolean {
  const intermediateStatuses = ['pre-creating', 'creating', 'pre-deleting', 'deleting', 'unknown']
  if (!vm.value) return true
  if (
    (tabId === 'backups' || tabId === 'interfaces') &&
    intermediateStatuses.includes(vm.value.status)
  ) {
    return true
  }
  return false
}

const navigateToTab = (tabPath: string) => {
  if (tabPath === '') {
    router.push(`/vm/${vmid.value}`)
  } else {
    router.push(`/vm/${vmid.value}/${tabPath}`)
  }
}

const vm = ref<VM | null>(null)

function fetchVMWithLoading() {
  loading.start('vm', vmid.value, 'fetch_vm')
  fetchVM().finally(() => {
    loading.stop('vm', vmid.value, 'fetch_vm')
  })
}

async function fetchVM() {
  try {
    const res = await api.get(`/vm/${vmid.value}`)
    vm.value = res.data as VM
  } catch (err) {
    console.error('Failed to fetch VM info:', err)
  }
}

function handleStatusChange(newStatus: string) {
  if (vm.value) {
    vm.value.status = newStatus
    fetchVM()
  }
}

let intervalId: number | null = null

onMounted(() => {
  fetchVMWithLoading()

  intervalId = setInterval(() => {
    fetchVM()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
  loading.clear('vm')
})
</script>

<template>
  <div>
    <div class="tabs tabs-lift">
      <template v-for="tab in tabs" :key="tab.id">
        <label class="tab" :class="{ 'tab-disabled': shouldDisableTab(tab.id) }">
          <input
            type="radio"
            name="vm_view_tabs"
            :checked="activeTab === tab.id"
            @change="navigateToTab(tab.path)"
          />
          <div class="flex items-center gap-2">
            <IconVue class="text-primary" :icon="tab.icon"></IconVue>
            <div>
              {{ tab.label }}
            </div>
          </div>
        </label>
        <div class="tab-content border-t border-t-black p-4">
          <div v-if="isLoading(vmid, 'fetch_vm')" class="grid h-70">
            <span class="loading loading-spinner place-self-center"></span>
          </div>
          <router-view
            v-if="!isLoading(vmid, 'fetch_vm') && vm"
            :vm="vm"
            @update-vm="fetchVM"
            @status-change="handleStatusChange"
          />
        </div>
      </template>
    </div>
  </div>
</template>
