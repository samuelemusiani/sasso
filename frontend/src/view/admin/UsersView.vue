<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'
import type { User } from '@/types'

const users = ref<User[]>([])
const isLoading = ref(true)
const searchQuery = ref('')
const expandedUser = ref<number | null>(null)
const editingUser = ref<{ [key: number]: User }>({})
const isUpdating = ref(false)

// Utenti filtrati per ricerca
const filteredUsers = computed(() => {
  if (!searchQuery.value) return users.value
  
  const query = searchQuery.value.toLowerCase()
  return users.value.filter(user => 
    user.username.toLowerCase().includes(query) ||
    user.email.toLowerCase().includes(query) ||
    user.realm.toLowerCase().includes(query)
  )
})

// Statistiche utenti
const userStats = computed(() => {
  const total = users.value.length
  const admins = users.value.filter(u => u.role === 'admin').length
  const realms = [...new Set(users.value.map(u => u.realm))].length
  
  return { total, admins, realms }
})

async function fetchUsers() {
  try {
    isLoading.value = true
    const response = await api.get('/admin/users')
    users.value = response.data as User[]
  } catch (error) {
    console.error('Errore nel caricamento utenti:', error)
  } finally {
    isLoading.value = false
  }
}

function getRoleIcon(role: string) {
  switch (role) {
    case 'admin': return 'material-symbols:admin-panel-settings'
    case 'user': return 'material-symbols:person'
    default: return 'material-symbols:help'
  }
}

function getRoleColor(role: string) {
  switch (role) {
    case 'admin': return 'text-error'
    case 'user': return 'text-primary'
    default: return 'text-base-content'
  }
}

function getRoleBadgeClass(role: string) {
  switch (role) {
    case 'admin': return 'badge-error'
    case 'user': return 'badge-primary'
    default: return 'badge-neutral'
  }
}

function toggleUserEdit(user: User) {
  if (expandedUser.value === user.id) {
    expandedUser.value = null
    delete editingUser.value[user.id]
  } else {
    expandedUser.value = user.id
    editingUser.value[user.id] = { ...user }
  }
}

async function updateUserLimits(userId: number) {
  try {
    isUpdating.value = true
    const userLimits = editingUser.value[userId]
    
    await api.put('/admin/users/limits', {
      user_id: userId,
      max_cores: userLimits.max_cores,
      max_ram: userLimits.max_ram,
      max_disk: userLimits.max_disk,
      max_nets: userLimits.max_nets,
    })
    
    // Aggiorna i dati locali
    const userIndex = users.value.findIndex(u => u.id === userId)
    if (userIndex !== -1) {
      users.value[userIndex] = { ...userLimits }
    }
    
    expandedUser.value = null
    delete editingUser.value[userId]
    
    // Mostra notifica di successo
    globalNotifications.showSuccess('Quote utente aggiornate con successo!')
  } catch (error) {
    console.error('Errore nell\'aggiornamento delle quote:', error)
    globalNotifications.showError('Errore nell\'aggiornamento delle quote')
  } finally {
    isUpdating.value = false
  }
}

onMounted(() => {
  fetchUsers()
})
</script>

<template>
  <div class="h-full overflow-auto">
    <!-- Header con breadcrumb -->
    <div class="mb-6 px-2">
      <div class="flex items-center gap-2 mb-2">
        <RouterLink to="/admin" class="btn btn-ghost btn-sm gap-2">
          <Icon icon="material-symbols:arrow-back" />
          Admin Panel
        </RouterLink>
        <span class="text-base-content/50">/</span>
        <span class="text-base-content font-medium">Gestione Utenti</span>
      </div>
      
      <div class="flex items-center gap-3 mb-4">
        <div class="btn btn-square btn-lg rounded-xl btn-primary p-0 flex-shrink-0">
          <Icon icon="mdi:account-group" class="text-2xl" />
        </div>
        <div>
          <h1 class="text-3xl font-bold text-base-content">Gestione Utenti</h1>
          <p class="text-base-content/70">Visualizza e modifica quote degli utenti registrati</p>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center items-center h-64">
      <div class="loading loading-spinner loading-lg"></div>
      <span class="ml-4 text-lg">Caricamento utenti...</span>
    </div>

    <div v-else>
      <!-- Statistiche utenti -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6 px-2">
        <div class="stat bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/20 rounded-xl">
          <div class="stat-figure text-blue-500">
            <Icon icon="mdi:account-group" class="text-3xl" />
          </div>
          <div class="stat-title text-blue-500/70">Totale Utenti</div>
          <div class="stat-value text-2xl text-blue-500">{{ userStats.total }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-red-500/10 to-red-600/10 border border-red-500/20 rounded-xl">
          <div class="stat-figure text-red-500">
            <Icon icon="material-symbols:admin-panel-settings" class="text-3xl" />
          </div>
          <div class="stat-title text-red-500/70">Amministratori</div>
          <div class="stat-value text-2xl text-red-500">{{ userStats.admins }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-purple-500/10 to-purple-600/10 border border-purple-500/20 rounded-xl">
          <div class="stat-figure text-purple-500">
            <Icon icon="material-symbols:domain-verification" class="text-3xl" />
          </div>
          <div class="stat-title text-purple-500/70">Realms Attivi</div>
          <div class="stat-value text-2xl text-purple-500">{{ userStats.realms }}</div>
        </div>
      </div>

      <!-- Barra di ricerca -->
      <div class="mb-6 px-2">
        <div class="form-control w-full max-w-md">
          <div class="input-group">
            <span class="bg-base-200">
              <Icon icon="material-symbols:search" class="text-xl" />
            </span>
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Cerca per username, email o realm..."
              class="input input-bordered w-full"
            />
          </div>
        </div>
      </div>

      <!-- Tabella utenti moderna -->
      <div class="px-2">
        <div class="card shadow-xl bg-base-100 border border-base-300">
          <div class="card-body p-0">
            <div class="overflow-x-auto">
              <table class="table table-zebra w-full">
                <thead>
                  <tr class="bg-base-200">
                    <th class="font-bold">
                      <Icon icon="material-symbols:tag" class="inline mr-2" />
                      ID
                    </th>
                    <th class="font-bold">
                      <Icon icon="material-symbols:person" class="inline mr-2" />
                      Utente
                    </th>
                    <th class="font-bold">
                      <Icon icon="material-symbols:email" class="inline mr-2" />
                      Email
                    </th>
                    <th class="font-bold">
                      <Icon icon="material-symbols:security" class="inline mr-2" />
                      Ruolo
                    </th>
                    <th class="font-bold">
                      <Icon icon="material-symbols:domain" class="inline mr-2" />
                      Realm
                    </th>
                    <th class="font-bold">
                      <Icon icon="material-symbols:settings" class="inline mr-2" />
                      Azioni
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <template v-for="user in filteredUsers" :key="user.id">
                    <tr class="hover">
                      <td class="font-mono text-sm">{{ user.id }}</td>
                      <td>
                        <div class="flex items-center gap-3">
                          <div class="w-10 h-10 bg-primary text-primary-content rounded-full flex items-center justify-center">
                            <span class="text-lg font-semibold">{{ user.username.charAt(0).toUpperCase() }}</span>
                          </div>
                          <div>
                            <div class="font-bold">{{ user.username }}</div>
                          </div>
                        </div>
                      </td>
                      <td class="text-sm text-base-content/70">{{ user.email }}</td>
                      <td>
                        <div class="flex items-center gap-2">
                          <Icon :icon="getRoleIcon(user.role)" :class="getRoleColor(user.role)" />
                          <span class="badge badge-sm" :class="getRoleBadgeClass(user.role)">
                            {{ user.role }}
                          </span>
                        </div>
                      </td>
                      <td>
                        <span class="badge badge-outline badge-sm">{{ user.realm }}</span>
                      </td>
                      <td>
                        <button 
                          @click="toggleUserEdit(user)"
                          class="btn btn-ghost btn-sm gap-2 hover:btn-primary"
                          :class="{ 'btn-primary': expandedUser === user.id }"
                        >
                          <Icon :icon="expandedUser === user.id ? 'material-symbols:expand-less' : 'material-symbols:edit'" />
                          {{ expandedUser === user.id ? 'Chiudi' : 'Modifica Quote' }}
                        </button>
                      </td>
                    </tr>
                    
                    <!-- Riga espandibile per modifica quote -->
                    <tr v-if="expandedUser === user.id" class="bg-base-200/50">
                      <td colspan="6" class="p-0">
                        <div class="p-6 border-t border-base-300">
                          <div class="flex items-center gap-3 mb-4">
                            <Icon icon="material-symbols:tune" class="text-2xl text-primary" />
                            <div>
                              <h3 class="font-bold text-lg">Modifica Quote per {{ user.username }}</h3>
                              <p class="text-sm text-base-content/70">Imposta i limiti massimi delle risorse per questo utente</p>
                            </div>
                          </div>
                          
                          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                            <!-- CPU Cores -->
                            <div class="form-control">
                              <label class="label">
                                <span class="label-text font-medium flex items-center gap-2">
                                  <Icon icon="material-symbols:memory" class="text-lg text-blue-500" />
                                  CPU Cores
                                </span>
                              </label>
                              <input
                                v-model.number="editingUser[user.id].max_cores"
                                type="number"
                                min="1"
                                max="64"
                                class="input input-bordered input-sm"
                                placeholder="Max cores"
                              />
                            </div>
                            
                            <!-- RAM (GB) -->
                            <div class="form-control">
                              <label class="label">
                                <span class="label-text font-medium flex items-center gap-2">
                                  <Icon icon="material-symbols:memory-alt" class="text-lg text-green-500" />
                                  RAM (GB)
                                </span>
                              </label>
                              <input
                                v-model.number="editingUser[user.id].max_ram"
                                type="number"
                                min="1"
                                max="256"
                                class="input input-bordered input-sm"
                                placeholder="Max RAM GB"
                              />
                            </div>
                            
                            <!-- Disk (GB) -->
                            <div class="form-control">
                              <label class="label">
                                <span class="label-text font-medium flex items-center gap-2">
                                  <Icon icon="material-symbols:storage" class="text-lg text-orange-500" />
                                  Disk (GB)
                                </span>
                              </label>
                              <input
                                v-model.number="editingUser[user.id].max_disk"
                                type="number"
                                min="10"
                                max="2048"
                                class="input input-bordered input-sm"
                                placeholder="Max disk GB"
                              />
                            </div>
                            
                            <!-- Networks -->
                            <div class="form-control">
                              <label class="label">
                                <span class="label-text font-medium flex items-center gap-2">
                                  <Icon icon="material-symbols:network-node" class="text-lg text-purple-500" />
                                  Reti
                                </span>
                              </label>
                              <input
                                v-model.number="editingUser[user.id].max_nets"
                                type="number"
                                min="1"
                                max="32"
                                class="input input-bordered input-sm"
                                placeholder="Max networks"
                              />
                            </div>
                          </div>
                          
                          <!-- Pulsanti azione -->
                          <div class="flex gap-3">
                            <button
                              @click="updateUserLimits(user.id)"
                              :disabled="isUpdating"
                              class="btn btn-primary btn-sm gap-2"
                            >
                              <Icon icon="material-symbols:save" />
                              {{ isUpdating ? 'Salvataggio...' : 'Salva Quote' }}
                            </button>
                            
                            <button
                              @click="toggleUserEdit(user)"
                              class="btn btn-ghost btn-sm gap-2"
                            >
                              <Icon icon="material-symbols:cancel" />  
                              Annulla
                            </button>
                          </div>
                        </div>
                      </td>
                    </tr>
                  </template>
                </tbody>
              </table>
            </div>
            
            <!-- Stato vuoto per ricerca -->
            <div v-if="filteredUsers.length === 0 && searchQuery" class="p-8 text-center">
              <Icon icon="material-symbols:search-off" class="text-6xl text-base-content/30 mb-4" />
              <p class="text-base-content/70">Nessun utente trovato per "{{ searchQuery }}"</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
