<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import type { AdminPortForward } from '@/types'

const isLoading = ref(true)
const users = ref([])
const realms = ref([])
const globalSSHKeys = ref([])
const portForwards = ref<AdminPortForward[]>([])

// Statistiche computate
const adminStats = computed(() => {
  const totalUsers = users.value.length
  const activeRealms = realms.value.length
  const totalSSHKeys = globalSSHKeys.value.length
  const totalPortForwards = portForwards.value.length
  const pendingPortForwards = portForwards.value.filter(pf => !pf.approved).length
  
  return { totalUsers, activeRealms, totalSSHKeys, totalPortForwards, pendingPortForwards }
})

// Sezioni admin con informazioni
const adminSections = computed(() => [
  {
    id: 'users',
    title: 'Gestione Utenti',
    description: 'Gestisci utenti, permessi e quote risorse',
    icon: 'mdi:account-group',
    route: '/admin/users',
    count: adminStats.value.totalUsers,
    countLabel: 'utenti registrati',
    color: 'from-blue-500/20 to-blue-600/20',
    borderColor: 'border-blue-500/30',
    iconColor: 'text-blue-500'
  },
  {
    id: 'realms',
    title: 'Domini Autenticazione',
    description: 'Configura LDAP, OAuth e altri metodi di auth',
    icon: 'material-symbols:domain-verification',
    route: '/admin/realms',
    count: adminStats.value.activeRealms,
    countLabel: 'realms configurati',
    color: 'from-purple-500/20 to-purple-600/20',
    borderColor: 'border-purple-500/30',
    iconColor: 'text-purple-500'
  },
  {
    id: 'ssh-keys',
    title: 'Chiavi SSH Globali',
    description: 'Gestisci chiavi SSH condivise per tutti gli utenti',
    icon: 'material-symbols:key',
    route: '/admin/ssh-keys',
    count: adminStats.value.totalSSHKeys,
    countLabel: 'chiavi globali',
    color: 'from-emerald-500/20 to-emerald-600/20',
    borderColor: 'border-emerald-500/30',
    iconColor: 'text-emerald-500'
  },
  {
    id: 'port-forwards',
    title: 'Port Forwarding',
    description: `${adminStats.value.totalPortForwards} totali • ${adminStats.value.pendingPortForwards} in attesa`,
    icon: 'material-symbols:router',
    route: '/admin/port-forwards',
    count: adminStats.value.pendingPortForwards,
    countLabel: 'richieste in attesa',
    color: 'from-orange-500/20 to-orange-600/20',
    borderColor: 'border-orange-500/30',
    iconColor: 'text-orange-500'
  }
])

async function fetchAdminData() {
  try {
    isLoading.value = true
    
    // Fetch dati parallelo
    const [usersRes, realmsRes, sshKeysRes, portForwardsRes] = await Promise.allSettled([
      api.get('/admin/users'),
      api.get('/admin/realms'),
      api.get('/admin/ssh-keys'),
      api.get('/admin/port-forwards')
    ])
    
    if (usersRes.status === 'fulfilled') {
      users.value = usersRes.value.data || []
    }
    if (realmsRes.status === 'fulfilled') {
      realms.value = realmsRes.value.data || []
    }
    if (sshKeysRes.status === 'fulfilled') {
      globalSSHKeys.value = sshKeysRes.value.data || []
    }
    if (portForwardsRes.status === 'fulfilled') {
      portForwards.value = portForwardsRes.value.data || []
    }
  } catch (error) {
    console.error('Errore nel caricamento dati admin:', error)
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  fetchAdminData()
})
</script>

<template>
  <!-- Contenuto principale della dashboard admin -->
  <div class="h-full overflow-auto">
    <!-- Header -->
    <div class="mb-8 px-2">
      <div class="flex items-center gap-3 mb-4">
        <div class="btn btn-square btn-lg rounded-xl btn-primary p-0 flex-shrink-0">
          <Icon icon="material-symbols:admin-panel-settings" class="text-2xl" />
        </div>
        <div>
          <h1 class="text-3xl font-bold text-base-content">Pannello Amministrazione</h1>
          <p class="text-base-content/70">Gestisci utenti, domini di autenticazione e configurazioni globali</p>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center items-center h-64">
      <div class="loading loading-spinner loading-lg"></div>
      <span class="ml-4 text-lg">Caricamento pannello admin...</span>
    </div>

    <!-- Griglia delle sezioni admin -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8 px-2">
      <RouterLink
        v-for="section in adminSections"
        :key="section.id"
        :to="section.route"
        class="card shadow-2xl border border-white/30 
               bg-gradient-to-br backdrop-blur-2xl backdrop-saturate-200
               hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25),0_0_20px_rgba(255,255,255,0.1)]
               hover:scale-[1.02] hover:border-white/40
               transition-all duration-300 ease-out
               overflow-hidden relative group
               before:absolute before:inset-0 before:bg-gradient-to-r 
               before:from-transparent before:via-white/5 before:to-transparent
               before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300"
        :class="[section.color, section.borderColor]"
      >
        <div class="card-body p-6">
          <!-- Header sezione -->
          <div class="flex items-start justify-between mb-4">
            <div class="flex items-center gap-3">
              <div class="btn btn-square btn-lg rounded-xl p-0 bg-white/10 border-white/20 hover:bg-white/20">
                <Icon :icon="section.icon" class="text-2xl" :class="section.iconColor" />
              </div>
              <div>
                <h3 class="font-bold text-xl text-base-content mb-1">{{ section.title }}</h3>
                <p class="text-sm text-base-content/70">{{ section.description }}</p>
              </div>
            </div>
          </div>

          <!-- Statistiche -->
          <div class="flex items-center justify-between mt-auto">
            <div class="flex items-center gap-2">
              <span class="text-3xl font-bold" :class="section.iconColor">{{ section.count }}</span>
              <span class="text-sm text-base-content/70">{{ section.countLabel }}</span>
            </div>
            
            <!-- Freccia di navigazione -->
            <div class="btn btn-square btn-sm rounded-lg bg-white/10 border-white/20 group-hover:bg-white/20 transition-colors">
              <Icon icon="material-symbols:arrow-forward" class="text-lg" />
            </div>
          </div>
        </div>

        <!-- Indicatore hover -->
        <div class="absolute bottom-0 left-0 w-full h-1 bg-gradient-to-r opacity-0 group-hover:opacity-100 transition-opacity duration-300"
             :class="section.iconColor.replace('text-', 'from-').replace('-500', '-400') + ' to-transparent'"></div>
      </RouterLink>
    </div>

    <!-- Sezione azioni rapide -->
    <div class="px-2 mb-6">
      <h2 class="text-xl font-bold text-base-content mb-4 flex items-center gap-2">
        <Icon icon="material-symbols:flash-on" class="text-warning" />
        Azioni Rapide
      </h2>
      
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <RouterLink to="/admin/realms?add=true" 
                   class="btn btn-outline btn-sm flex-col gap-2 p-4 h-auto hover:bg-purple-500/10">
          <Icon icon="material-symbols:add-circle" class="text-2xl text-purple-500" />
          <span class="text-xs">Nuovo Realm</span>
        </RouterLink>
        
        <RouterLink to="/admin/ssh-keys" 
                   class="btn btn-outline btn-sm flex-col gap-2 p-4 h-auto hover:bg-emerald-500/10">
          <Icon icon="material-symbols:vpn-key" class="text-2xl text-emerald-500" />
          <span class="text-xs">Aggiungi Chiave</span>
        </RouterLink>
        
        <RouterLink to="/admin/port-forwards?create=true" 
                   class="btn btn-outline btn-sm flex-col gap-2 p-4 h-auto hover:bg-orange-500/10">
          <Icon icon="material-symbols:add-box" class="text-2xl text-orange-500" />
          <span class="text-xs">Crea Port Forward</span>
        </RouterLink>
      </div>
    </div>
  </div>
</template>
