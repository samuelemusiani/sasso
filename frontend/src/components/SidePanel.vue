<script setup lang="ts">
import PanelLine from '@/components/PanelLine.vue'
import { useRouter } from 'vue-router'
import { onMounted, computed, ref } from 'vue'
import type { User } from '@/types'
import { api } from '@/lib/api'

const collapsed = ref(false)
const router = useRouter()

const menu = {
  Home: { icon: 'material-symbols:home-rounded', to: '/' },
  'Virtual Machine': { icon: 'mi:computer', to: '/vm' },
  Nets: { icon: 'ph:network', to: '/net' },
  'SSH Keys': { icon: 'icon-park-twotone:key', to: '/ssh-keys' },
  VPN: { icon: 'cib:wireguard', to: '/vpn' },
  'Port Forward': { icon: 'material-symbols:router', to: '/port-forwards' },
  'Telegram Bots': { icon: 'mdi:telegram', to: '/telegram' },
}

function logout() {
  localStorage.removeItem('jwt_token')
  router.push('/login')
}

const showAdminPanel = computed(() => {
  if (!whoami.value) return false
  return whoami.value.role === 'admin'
})
const whoami = ref<User | null>(null)

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      whoami.value = res.data as User
    })
    .catch((err) => {
      console.error('Failed to fetch whoami:', err)
    })
}

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
        collapsed ? 'w-16' : 'w-54',
      ]"
    >
      <!-- Toggle -->
      <div class="flex w-full items-center justify-between" :class="{ 'flex-col': collapsed }">
        <a href="/" class="px-3 transition" :class="{ '!p-0': collapsed }">
          <img
            :src="collapsed ? '/sasso-icon.png' : '/sasso.png'"
            alt="Sasso Logo"
            :class="{ 'my-1 h-8': collapsed }"
          />
        </a>
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
        <li v-for="(item, name) in menu" :key="name">
          <PanelLine :to="item.to" :icon="item.icon" :label="name" :collapsed="collapsed" />
        </li>
        <div class="divider" v-if="showAdminPanel"></div>
        <li v-if="showAdminPanel">
          <PanelLine
            to="/admin"
            icon="material-symbols:admin-panel-settings"
            label="Admin Panel"
            :collapsed="collapsed"
          />
        </li>
      </ul>

      <!-- Footer actions -->
      <!-- TODO: user avatar for user settings -->
      <div class="border-base-300 w-full border-t p-2">
        <PanelLine
          to="/settings"
          icon="material-symbols:settings"
          label="Settings"
          :collapsed="collapsed"
        />
        <button
          @click="logout()"
          class="btn hover:bg-error-content flex w-full items-center gap-2 rounded-full font-semibold"
          :class="{ '!justify-center !rounded-2xl': collapsed }"
        >
          <IconVue icon="material-symbols:logout" class="text-xl" />
          <span v-if="!collapsed">Logout</span>
        </button>
      </div>
    </div>
  </div>
</template>
