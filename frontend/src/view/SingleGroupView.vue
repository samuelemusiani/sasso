<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getPageIcon } from '@/const'
import { api } from '@/lib/api'
import type { Group, GroupMember } from '@/types'
import { useToastService } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const { error: toastError } = useToastService()

const tabs = [
  { id: 'info', label: 'Info', path: '', icon: `${getPageIcon('info')}` },
  { id: 'resources', label: 'Resources', path: 'resources', icon: `${getPageIcon('resources')}` },
  {
    id: 'members',
    label: 'Members',
    path: 'members',
    icon: `${getPageIcon('group')}`,
  },
]

const groupid = computed(() => {
  const id = route.params.id
  if (Array.isArray(id)) return Number(id[0])
  return id ? Number(id) : 0
})

const activeTab = computed(() => {
  const path = route.path
  for (const tab of tabs) {
    if (tab.path === '') {
      // Check if we're at the base path /vm/:vmid
      if (path === `/group/${groupid.value}`) return tab.id
    } else {
      // Special case for backups, which has a sub-route for requests
      if (path.endsWith(`/${tab.path}`) || path.endsWith(`/${tab.path}/requests`)) return tab.id
    }
  }
  return 'info'
})

const navigateToTab = (tabPath: string) => {
  if (tabPath === '') {
    router.push(`/group/${groupid.value}`)
  } else {
    router.push(`/group/${groupid.value}/${tabPath}`)
  }
}

const group = ref<Group | null>(null)
const me = ref<GroupMember | null>(null)

function fetchGroup() {
  api
    .get(`/groups/${groupid.value}`)
    .then((res) => {
      group.value = res.data as Group
    })
    .catch((err) => {
      console.error('Failed to fetch Group:', err)
      toastError(`Failed to fetch Group. ${err.response?.data}`)
    })
}

function fetchMe() {
  api
    .get(`/groups/${groupid.value}/members/me`)
    .then((res) => {
      me.value = res.data as GroupMember
    })
    .catch((err) => {
      console.error('Failed to fetch current user membership:', err)
    })
}

function fetchMembers() {
  api
    .get(`/groups/${groupid.value}/members`)
    .then((res) => {
      const tmp = res.data.sort((a: GroupMember, b: GroupMember) => a.user_id - b.user_id)
      if (group.value) {
        group.value.members = tmp as GroupMember[]
      }
    })
    .catch((err) => {
      console.error('Failed to fetch Members:', err)
    })
}

onMounted(() => {
  fetchGroup()
  fetchMe()
})
</script>

<template>
  <div class="relative p-2">
    <HelpButton class="absolute right-2" />
    <div class="tabs tabs-lift">
      <template v-for="tab in tabs" :key="tab.id">
        <label class="tab">
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
        <div class="tab-content border-t-base-300 border-t pt-4">
          <!-- <div v-if="isLoading(vmid, 'fetch_vm')" class="grid h-70"> -->
          <div v-if="false" class="grid h-70">
            <span class="loading loading-spinner text-primary place-self-center"></span>
          </div>
          <template v-else-if="group && me && activeTab === tab.id">
            <!-- <router-view :vm="vm" @update-vm="fetchVM" @status-change="handleStatusChange" /> -->
            <router-view
              :group="group"
              :me="me"
              @fetch-group="fetchGroup"
              @fetch-members="fetchMembers"
            />
          </template>
        </div>
      </template>
    </div>
  </div>
</template>
