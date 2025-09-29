<script setup lang="ts">
import DrawerLine from '@/components/DrawerLine.vue'
import { useRouter } from 'vue-router'
import { onMounted, computed, ref } from 'vue'
import type { User } from '@/types'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'

const collapsed = ref(false)
const router = useRouter()

const menu = {
  Home: { icon: 'material-symbols:home-rounded', to: '/' },
  VM: { icon: 'mi:computer', to: '/vm' },
  Nets: { icon: 'ph:network', to: '/net' },
  'SSH Keys': { icon: 'icon-park-twotone:key', to: '/ssh-keys' },
  VPN: { icon: 'cib:wireguard', to: '/vpn' },
  'Port Forward': { icon: 'material-symbols:router', to: '/port-forward' },
}

function logout() {
  localStorage.removeItem('jwt_token')
  router.push('/login')
}

const showAdminPanel = computed(() => {
  if (!whoami.value) return false
  return whoami.value.role === 'admin'
})

const userDisplayName = computed(() => {
  if (!whoami.value?.username) return 'User'
  return (
    whoami.value.username.charAt(0).toUpperCase() + whoami.value.username.slice(1).toLowerCase()
  )
})

const userInitial = computed(() => {
  return whoami.value?.username ? whoami.value.username[0].toUpperCase() : 'U'
})

const whoami = ref<User | null>(null)
const userAvatar = ref<string | null>(null)

function fetchWhoami() {
  api
    .get('/whoami')
    .then((res) => {
      console.log('Whoami response:', res.data)
      whoami.value = res.data as User

      // Tenta di caricare l'avatar se l'utente ha un'immagine LDAP
      if (res.data.avatar_url) {
        userAvatar.value = res.data.avatar_url
      } else if (res.data.email) {
        // Fallback: tenta di caricare da Gravatar usando l'email
        tryLoadGravatar(res.data.email)
      }
    })
    .catch((err) => {
      console.error('Failed to fetch whoami:', err)
    })
}

function tryLoadGravatar(email: string) {
  // Crea hash MD5 dell'email per Gravatar (semplificato)
  const emailHash = btoa(email.toLowerCase().trim()).replace(/[^a-zA-Z0-9]/g, '')
  const gravatarUrl = `https://www.gravatar.com/avatar/${emailHash}?s=40&d=404`

  // Testa se l'immagine Gravatar esiste
  const img = new Image()
  img.onload = () => {
    userAvatar.value = gravatarUrl
  }
  img.onerror = () => {
    userAvatar.value = null // Usa fallback con lettera
  }
  img.src = gravatarUrl
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
        'bg-linear-to-r from-base-200 to-base-200/40 backdrop-blur-md flex flex-col transition-all duration-300 shadow-lg items-center my-4 rounded-xl',
        collapsed ? 'w-16' : 'w-64',
      ]"
    >
      <!-- Toggle -->
      <div class="flex justify-between items-center w-full px-4" :class="{ 'flex-col': collapsed }">
        <a href="/" class="px-3 transition" :class="{ '!p-0': collapsed }">
          <img
            :src="collapsed ? '/logo-sasso.png' : '/sasso.png'"
            alt="Sasso Logo"
            :class="{ 'h-8 my-1': collapsed }"
          />
        </a>
        <button
          class="btn btn-ghost btn-sm m-2 text-xl"
          @click="collapsed = !collapsed"
          :title="collapsed ? 'Expand' : 'Collapse'"
        >
          <Icon
            :icon="
              collapsed
                ? 'material-symbols:chevron-right-rounded'
                : 'material-symbols:chevron-left-rounded'
            "
          />
        </button>
      </div>

      <!-- Menu -->
      <ul class="menu flex-1 gap-1 w-full">
        <li v-for="(item, name) in menu" :key="name">
          <DrawerLine :to="item.to" :icon="item.icon" :label="name" :collapsed="collapsed" />
        </li>
        <!-- Admin Panel - solo per utenti admin -->
        <div class="divider" v-if="showAdminPanel"></div>
        <li v-if="showAdminPanel">
          <DrawerLine
            to="/admin"
            icon="material-symbols:admin-panel-settings"
            label="Admin Panel"
            :collapsed="collapsed"
          ></DrawerLine>
        </li>
      </ul>

      <!-- Footer actions -->
      <div class="p-2 border-t border-base-300 w-full">
        <DrawerLine
          to="/settings"
          icon="material-symbols:settings"
          label="Settings"
          :collapsed="collapsed"
        />

        <!-- User Profile Section -->
        <div class="p-2 mt-2">
          <div
            v-if="!collapsed"
            class="flex items-center justify-between bg-base-200/50 rounded-lg p-3"
          >
            <!-- User Info -->
            <div class="flex items-center gap-3">
              <!-- Avatar -->
              <div class="avatar">
                <div
                  class="w-8 h-8 rounded-full overflow-hidden bg-primary flex items-center justify-center"
                >
                  <img
                    v-if="userAvatar"
                    :src="userAvatar"
                    :alt="userDisplayName"
                    class="w-full h-full object-cover"
                    @error="userAvatar = null"
                  />
                  <span
                    v-else
                    class="text-primary-content text-lg font-bold leading-none flex items-center justify-center w-full h-full"
                  >
                    {{ userInitial }}
                  </span>
                </div>
              </div>
              <!-- Name -->
              <div class="flex flex-col">
                <span class="text-sm font-medium text-base-content">
                  {{ userDisplayName }}
                </span>
                <span class="text-xs text-base-content/60 capitalize">
                  {{ whoami?.role || 'user' }}
                </span>
              </div>
            </div>

            <!-- Logout Button -->
            <button
              @click="logout()"
              class="btn btn-ghost btn-sm text-base-content/70 hover:text-error hover:bg-error/10"
              title="Logout"
            >
              <IconifyIcon icon="material-symbols:logout" class="text-lg" />
            </button>
          </div>

          <!-- Collapsed version -->
          <div v-else class="flex flex-col gap-2 items-center">
            <!-- Avatar -->
            <div class="avatar">
              <div
                class="w-8 h-8 rounded-full overflow-hidden bg-primary flex items-center justify-center"
              >
                <img
                  v-if="userAvatar"
                  :src="userAvatar"
                  :alt="userDisplayName"
                  class="w-full h-full object-cover"
                  @error="userAvatar = null"
                />
                <span
                  v-else
                  class="text-primary-content text-base font-bold leading-none flex items-center justify-center w-full h-full"
                >
                  {{ userInitial }}
                </span>
              </div>
            </div>
            <!-- Logout Button -->
            <button
              @click="logout()"
              class="btn btn-ghost btn-sm text-base-content/70 hover:text-error hover:bg-error/10"
              title="Logout"
            >
              <IconifyIcon icon="material-symbols:logout" class="text-lg" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
