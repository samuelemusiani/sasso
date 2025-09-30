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
  'Port Forward': { icon: 'material-symbols:router', to: '/port-forwards' }
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
      console.log('Whoami response:', res.data)
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
    <div :class="[
      'bg-linear-to-r from-base-200 to-base-200/40 backdrop-blur-md flex flex-col transition-all duration-300 shadow-lg items-center my-4 rounded-xl',
      collapsed ? 'w-16' : 'w-54',
    ]">
      <!-- Toggle -->
      <div class="flex justify-between items-center w-full" :class="{ 'flex-col': collapsed }">
        <a href="/" class="px-3 transition" :class="{ '!p-0': collapsed }">
          <img :src="collapsed ? '/sasso-icon.png' : '/sasso.png'" alt="Sasso Logo"
            :class="{ 'h-8 my-1': collapsed }" />
        </a>
        <button class="btn btn-ghost btn-sm m-2 text-xl" @click="collapsed = !collapsed"
          :title="collapsed ? 'Expand' : 'Collapse'">
          <IconVue :icon="collapsed
              ? 'material-symbols:chevron-right-rounded'
              : 'material-symbols:chevron-left-rounded'
            " />
        </button>
      </div>

      <!-- Menu -->
      <ul class="menu flex-1 gap-1 w-full">
        <li v-for="(item, name) in menu" :key="name">
          <PanelLine :to="item.to" :icon="item.icon" :label="name" :collapsed="collapsed" />
        </li>
        <div class="divider" v-if="showAdminPanel"></div>
        <li v-if="showAdminPanel">
          <PanelLine to="/admin" icon="material-symbols:admin-panel-settings" label="Admin Panel" :collapsed="collapsed" />
        </li>
      </ul>

      <!-- Footer actions -->
      <!-- TODO: user avatar for user settings -->
      <div class="p-2 border-t border-base-300 w-full">
        <PanelLine to="/settings" icon="material-symbols:settings" label="Settings" :collapsed="collapsed" />
        <button @click="logout()"
          class="flex items-center gap-2 p-2 hover:bg-primary/20 rounded-full w-full font-semibold"
          :class="{ '!justify-center !rounded-2xl': collapsed }">
          <IconVue icon="material-symbols:logout" class="text-xl ml-5" />
          <span v-if="!collapsed">Logout</span>
        </button>
      </div>
    </div>
  </div>
</template>
