<script setup lang="ts">
import PanelLine from '@/components/PanelLine.vue'
import { useRouter, RouterLink } from 'vue-router'
import { onMounted, computed, ref } from 'vue'
import type { User } from '@/types'
import { api } from '@/lib/api'
import { getPageIcon } from '@/const'

const collapsed = ref(false)
const router = useRouter()

function logout() {
  localStorage.removeItem('jwt_token')
  router.push('/login')
}

const whoami = ref<User | null>(null)

const showAdminPanel = computed(() => {
  if (!whoami.value) return false
  return whoami.value.role === 'admin'
})

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      whoami.value = res.data as User
    })
    .catch((err) => {
      if (err.response && err.response.status === 401) {
        // Could be first load and user is not logged in, ignore the error
        return
      }
      console.error('Failed to fetch whoami:', err)
    })
}

function setIconOnMenuItem(item: { label: string; to: string }) {
  return {
    ...item,
    icon: getPageIcon(item.to.substring(1)),
  }
}

type MenuItem = {
  label: string
  to: string
  icon: string
}

const topMenu: MenuItem[] = [
  { label: 'Home', to: '/home' },
  { label: 'Virtual Machine', to: '/vm' },
  { label: 'Nets', to: '/net' },
  { label: 'Interfaces', to: '/interfaces' },
  { label: 'SSH Keys', to: '/ssh-keys' },
  { label: 'VPN', to: '/vpn' },
  { label: 'Port Forward', to: '/port-forwards' },
  { label: 'Telegram Bots', to: '/telegram' },
  { label: 'Groups', to: '/group' },
].map(setIconOnMenuItem)

const middleMenu = computed(() => {
  const baseMiddleMenu: MenuItem[] = []
  baseMiddleMenu.map(setIconOnMenuItem)

  if (!showAdminPanel.value) return baseMiddleMenu
  const adminMenuItem = [{ label: 'Admin Panel', to: '/admin' }].map(setIconOnMenuItem)[0]
  return [...baseMiddleMenu, adminMenuItem]
})

const bottomMenu: MenuItem[] = [
  { label: 'Help', to: '/help' },
  { label: 'Settings', to: '/settings' },
].map(setIconOnMenuItem)

onMounted(() => {
  fetchWhoami()
})
</script>

<template>
  <div class="flex">
    <!-- Sidebar -->
    <div
      :class="[
        'from-base-200 to-base-200/40 my-4 flex flex-col items-center rounded-xl bg-linear-to-r shadow-lg backdrop-blur-md transition-all duration-300',
        collapsed ? 'w-16' : 'w-56',
      ]"
    >
      <!-- Toggle -->
      <div class="flex w-full items-center justify-between" :class="{ 'flex-col': collapsed }">
        <RouterLink to="/" class="px-3 transition" :class="{ 'p-0!': collapsed }">
          <img
            :src="collapsed ? '/sasso-icon.png' : '/sasso.png'"
            alt="Sasso Logo"
            :class="{ 'my-1 h-8': collapsed }"
          />
        </RouterLink>
        <button
          class="btn btn-ghost btn-sm m-2 text-xl"
          @click="collapsed = !collapsed"
          :title="collapsed ? 'Expand' : 'Collapse'"
        >
          <IconVue
            :icon="
              collapsed
                ? 'material-symbols:chevron-right-rounded'
                : 'material-symbols:chevron-left-rounded'
            "
          />
        </button>
      </div>

      <!-- Menu -->
      <ul class="menu w-full flex-1 gap-1">
        <li v-for="i in topMenu" :key="i.to">
          <PanelLine :to="i.to" :icon="i.icon" :label="i.label" :collapsed="collapsed" />
        </li>

        <div class="divider" v-if="middleMenu.length > 0"></div>

        <li v-for="i in middleMenu" :key="i.to">
          <PanelLine :to="i.to" :icon="i.icon" :label="i.label" :collapsed="collapsed" />
        </li>
      </ul>

      <!-- Footer actions -->
      <!-- TODO: user avatar for user settings -->
      <div class="border-base-300 flex w-full flex-col gap-2 border-t p-2">
        <template v-for="i in bottomMenu" :key="i.to">
          <PanelLine :to="i.to" :icon="i.icon" :label="i.label" :collapsed="collapsed" />
        </template>
        <button
          @click="logout()"
          class="btn hover:bg-error-content flex w-full items-center gap-2 rounded-full font-semibold"
          :class="{ 'justify-center! rounded-2xl!': collapsed }"
        >
          <IconVue icon="material-symbols:logout" class="text-xl" />
          <span v-if="!collapsed">Logout</span>
        </button>
      </div>
    </div>
  </div>
</template>
