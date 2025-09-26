<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { RouterLink } from 'vue-router'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'
import type { AdminPortForward } from '@/types'

const route = useRoute()
const portForwards = ref<AdminPortForward[]>([])
const isLoading = ref(true)
const searchQuery = ref('')

// Opzioni per il filtro stato
const statusOptions = [
  { value: 'all', label: 'Tutti', icon: 'material-symbols:list', color: 'text-base-content' },
  { value: 'approved', label: 'Approvati', icon: 'material-symbols:check-circle', color: 'text-success' },
  { value: 'pending', label: 'In attesa', icon: 'material-symbols:schedule', color: 'text-warning' }
]
const selectedStatus = ref('all')
const addingPortForward = ref(false)
const isProcessing = ref(false)

const newPortForward = ref<{
  dest_ip: string
  dest_port: string
}>({
  dest_ip: '',
  dest_port: ''
})

// Port forwards filtrati per ricerca (rimuoviamo i filtri di stato)
const filteredPortForwards = computed(() => {
  let filtered = portForwards.value

  // Filtro per testo di ricerca
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(pf => 
      pf.dest_ip.toLowerCase().includes(query) ||
      pf.dest_port.toString().includes(query) ||
      (pf.out_port && pf.out_port.toString().includes(query))
    )
  }

  return filtered
})

// Statistiche dei port forwarding
const pfStats = computed(() => ({
  totalRequests: portForwards.value.length,
  approvedRequests: portForwards.value.filter(pf => pf.approved).length,
  pendingRequests: portForwards.value.filter(pf => !pf.approved).length
}))

function fetchPortForwards() {
  isLoading.value = true
  api
    .get('/admin/port-forwards')
    .then((res) => {
      portForwards.value = res.data as AdminPortForward[]
    })
    .catch((err) => {
      console.error('Failed to fetch Port Forwards:', err)
      globalNotifications.showError('Errore nel caricamento dei port forwarding')
      portForwards.value = []
    })
    .finally(() => {
      isLoading.value = false
    })
}

function createPortForward() {
  if (!newPortForward.value.dest_ip || !newPortForward.value.dest_port) {
    globalNotifications.showError('Compila tutti i campi obbligatori')
    return
  }

  // Validazione IP
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/
  if (!ipRegex.test(newPortForward.value.dest_ip)) {
    globalNotifications.showError('Inserisci un indirizzo IP valido')
    return
  }

  // Validazione porta
  const destPort = parseInt(newPortForward.value.dest_port)
  
  if (destPort < 1 || destPort > 65535) {
    globalNotifications.showError('La porta deve essere compresa tra 1 e 65535')
    return
  }

  isProcessing.value = true
  
  // L'admin crea il port forward e poi lo approva automaticamente
  api
    .post('/port-forwards', {
      dest_ip: newPortForward.value.dest_ip,
      dest_port: destPort
    })
    .then(async (response) => {
      console.log('Created port forward:', response.data)
      
      // Dopo aver creato il port forward, lo approviamo automaticamente se abbiamo l'ID
      const newPortForwardId = response.data?.id
      if (newPortForwardId) {
        console.log('Approving port forward with ID:', newPortForwardId)
        await api.put(`/admin/port-forwards/${newPortForwardId}`, { approve: true })
        globalNotifications.showSuccess('Port forwarding creato e approvato con successo!')
      } else {
        // Fallback: refresh la lista e trova l'ultimo port forward per approvarlo
        console.log('No ID received, refreshing list to find and approve the new port forward')
        await fetchPortForwards()
        
        // Trova il port forward più recente per questo IP e porta e approvalo
        const latestPf = portForwards.value
          .filter(pf => pf.dest_ip === newPortForward.value.dest_ip && pf.dest_port === destPort && !pf.approved)
          .sort((a, b) => b.id - a.id)[0]
        
        if (latestPf) {
          await api.put(`/admin/port-forwards/${latestPf.id}`, { approve: true })
          globalNotifications.showSuccess('Port forwarding creato e approvato con successo!')
        } else {
          globalNotifications.showSuccess('Port forwarding creato con successo!')
        }
      }
      
      fetchPortForwards()
      newPortForward.value = {
        dest_ip: '',
        dest_port: ''
      }
      addingPortForward.value = false
    })
    .catch((err) => {
      console.error('Failed to add port forward:', err)
      globalNotifications.showError('Errore nella creazione del port forwarding')
    })
    .finally(() => {
      isProcessing.value = false
    })
}

function approvePortForward(id: number) {
  console.log('Approving port forward:', id)
  
  api
    .put(`/admin/port-forwards/${id}`, { approve: true })
    .then(() => {
      fetchPortForwards()
      globalNotifications.showSuccess('Port forwarding approvato!')
    })
    .catch((err) => {
      console.error('Failed to approve port forward:', err)
      globalNotifications.showError('Errore nell\'approvazione del port forwarding')
    })
}

function deletePortForward(id: number) {
  if (!confirm('Sei sicuro di voler eliminare questo port forwarding?')) {
    return
  }

  api
    .delete(`/port-forwards/${id}`)
    .then(() => {
      fetchPortForwards()
      globalNotifications.showSuccess('Port forwarding eliminato!')
    })
    .catch((err) => {
      console.error('Failed to delete port forward:', err)
      globalNotifications.showError('Errore nell\'eliminazione del port forwarding')
    })
}

function getApprovalBadge(approved: boolean) {
  if (approved) {
    return { class: 'badge-success', icon: 'material-symbols:check-circle', text: 'Approvato' }
  } else {
    return { class: 'badge-warning', icon: 'material-symbols:schedule', text: 'In attesa' }
  }
}

onMounted(() => {
  fetchPortForwards()
  
  // Controlla se deve aprire automaticamente il form di aggiunta
  if (route.query.add === 'true') {
    addingPortForward.value = true
  }
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
        <span class="text-base-content font-medium">Port Forwarding</span>
      </div>
      
      <div class="flex items-center gap-3 mb-4">
        <div class="btn btn-square btn-lg rounded-xl btn-primary p-0 flex-shrink-0">
          <Icon icon="material-symbols:router" class="text-2xl" />
        </div>
        <div>
          <h1 class="text-3xl font-bold text-base-content">Gestione Port Forwarding</h1>
          <p class="text-base-content/70">Gestisci le richieste di port forwarding degli utenti</p>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center items-center h-64">
      <div class="loading loading-spinner loading-lg"></div>
      <span class="ml-4 text-lg">Caricamento port forwards...</span>
    </div>

    <div v-else>

      <!-- Statistiche e controlli -->
      <div class="px-2 mb-6">
        <div class="card shadow-xl bg-base-100 border border-base-300">
          <div class="card-body">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
              <div class="stat bg-gradient-to-br from-primary/10 to-primary/20 border border-primary/20 rounded-xl">
                <div class="stat-figure text-primary">
                  <Icon icon="material-symbols:list" class="text-3xl" />
                </div>
                <div class="stat-title text-primary/70">Totali</div>
                <div class="stat-value text-2xl text-primary">{{ pfStats.totalRequests }}</div>
              </div>
              
              <div class="stat bg-gradient-to-br from-warning/10 to-warning/20 border border-warning/20 rounded-xl">
                <div class="stat-figure text-warning">
                  <Icon icon="material-symbols:schedule" class="text-3xl" />
                </div>
                <div class="stat-title text-warning/70">In attesa</div>
                <div class="stat-value text-2xl text-warning">{{ pfStats.pendingRequests }}</div>
              </div>
              
              <div class="stat bg-gradient-to-br from-success/10 to-success/20 border border-success/20 rounded-xl">
                <div class="stat-figure text-success">
                  <Icon icon="material-symbols:check-circle" class="text-3xl" />
                </div>
                <div class="stat-title text-success/70">Approvati</div>
                <div class="stat-value text-2xl text-success">{{ pfStats.approvedRequests }}</div>
              </div>
            </div>
            
            <!-- Filtri e controlli -->
            <div class="flex flex-col md:flex-row gap-4 items-center justify-between">
              <!-- Ricerca -->
              <div class="flex-1 max-w-md">
                <div class="relative">
                  <Icon icon="material-symbols:search" class="absolute left-3 top-1/2 transform -translate-y-1/2 text-base-content/50" />
                  <input v-model="searchQuery" 
                         type="text" 
                         placeholder="Cerca per IP o porta..." 
                         class="input input-bordered pl-10 w-full" />
                </div>
              </div>
              
              <!-- Filtro Stato e pulsanti azione -->
              <div class="flex gap-2 shrink-0">
                <!-- Filtri stato -->
                <button v-for="status in statusOptions" 
                        :key="status.value"
                        @click="selectedStatus = status.value"
                        class="btn btn-sm gap-2 transition-all duration-200"
                        :class="selectedStatus === status.value ? 'btn-primary' : 'btn-ghost'">
                  <Icon :icon="status.icon" class="text-sm" :class="status.color" />
                  {{ status.label }}
                </button>
                
                <!-- Pulsante aggiungi port forward -->
                <button
                  @click="addingPortForward = true"
                  v-show="!addingPortForward"
                  class="btn btn-primary gap-2 h-12"
                >
                  <Icon icon="material-symbols:add" />
                  Nuovo Port Forward
                </button>
                
                <!-- Pulsante annulla -->
                <button
                  @click="addingPortForward = false"
                  v-show="addingPortForward"
                  class="btn btn-error gap-2 h-12"
                >
                  <Icon icon="material-symbols:cancel" />
                  Annulla
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Form aggiunta port forward -->
      <div v-if="addingPortForward" class="mb-6 px-2">
        <div class="card shadow-xl bg-base-100 border border-base-300">
          <div class="card-body">
            <div class="flex items-center gap-3 mb-4">
              <Icon icon="material-symbols:add-circle" class="text-3xl text-primary" />
              <div>
                <h3 class="font-bold text-xl">Aggiungi Nuovo Port Forward</h3>
                <p class="text-sm text-base-content/70">Crea una nuova regola di port forwarding (sarà automaticamente approvata)</p>
              </div>
            </div>
            
            <form @submit.prevent="createPortForward" class="space-y-4">
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <!-- IP Destinazione -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text font-medium">IP Destinazione *</span>
                  </label>
                  <input
                    v-model="newPortForward.dest_ip"
                    type="text"
                    placeholder="192.168.1.100"
                    class="input input-bordered w-full"
                    pattern="^(\d{1,3}\.){3}\d{1,3}$"
                    required
                  />
                </div>
                
                <!-- Porta Destinazione -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text font-medium">Porta Destinazione *</span>
                  </label>
                  <input
                    v-model="newPortForward.dest_port"
                    type="number"
                    placeholder="8080"
                    class="input input-bordered w-full"
                    min="1"
                    max="65535"
                    required
                  />
                </div>
              </div>
              
              <!-- Azioni -->
              <div class="flex justify-end gap-2 pt-4">
                <button
                  type="button"
                  @click="addingPortForward = false"
                  class="btn btn-ghost"
                >
                  Annulla
                </button>
                <button
                  type="submit"
                  :disabled="isProcessing"
                  class="btn btn-primary gap-2"
                >
                  <span v-if="isProcessing" class="loading loading-spinner loading-sm"></span>
                  <Icon v-else icon="material-symbols:add" />
                  {{ isProcessing ? 'Creazione...' : 'Crea Port Forward' }}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>

      <!-- Lista Port Forwards -->
      <div v-show="!addingPortForward" class="px-2">
        <div class="card shadow-xl bg-base-100 border border-base-300">
          <div class="card-body p-0">
        <div class="p-6 border-b border-base-300/50">
          <h2 class="text-xl font-bold flex items-center gap-2">
            <Icon icon="material-symbols:list" class="text-primary" />
            Port Forwarding Attivi
          </h2>
        </div>
        
        <div class="overflow-x-auto">
          <!-- Loading State -->
          <div v-if="isLoading" class="flex justify-center items-center h-64">
            <div class="loading loading-spinner loading-lg"></div>
            <span class="ml-4 text-lg">Caricamento...</span>
          </div>
          
          <!-- Tabella port forwards -->
          <table v-else-if="filteredPortForwards.length > 0" class="table table-zebra w-full">
            <thead>
              <tr class="bg-base-200/50">
                <th class="font-semibold">ID</th>
                <th class="font-semibold">Utente</th>
                <th class="font-semibold">Porta Esterna</th>
                <th class="font-semibold">Destinazione</th>
                <th class="font-semibold">Stato</th>
                <th class="font-semibold">Azioni</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="pf in filteredPortForwards" :key="pf.id" class="hover">
                <td>
                  <div class="font-mono text-sm font-medium">
                    {{ pf.id }}
                  </div>
                </td>
                <td>
                  <div class="font-medium text-sm">
                    {{ pf.username }}
                  </div>
                </td>
                <td>
                  <div class="font-mono text-sm font-medium">
                    :{{ pf.out_port }}
                  </div>
                </td>
                <td>
                  <div class="font-mono text-sm">
                    {{ pf.dest_ip }}:{{ pf.dest_port }}
                  </div>
                </td>
                <td>
                  <div class="badge gap-1" :class="getApprovalBadge(pf.approved).class">
                    <Icon :icon="getApprovalBadge(pf.approved).icon" class="text-xs" />
                    {{ getApprovalBadge(pf.approved).text }}
                  </div>
                </td>
                <td>
                  <div class="flex gap-2">
                    <button v-if="!pf.approved" @click="approvePortForward(pf.id)" class="btn btn-success btn-sm gap-1">
                      <Icon icon="material-symbols:check" /> Approva
                    </button>
                    <button
                      @click="deletePortForward(pf.id)"
                      class="btn btn-error btn-sm gap-1"
                      title="Elimina port forwarding"
                    >
                      <Icon icon="material-symbols:delete" />
                      Elimina
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          
          <!-- Stato vuoto -->
          <div v-else class="p-8 text-center">
            <Icon icon="material-symbols:router-outline" class="text-6xl text-base-content/30 mb-4" />
            <p class="text-lg font-medium text-base-content/70 mb-2">Nessun port forwarding trovato</p>
            <p class="text-base-content/50">Utilizza il pulsante "Nuovo Port Forward" per crearne uno</p>
          </div>
        </div>
      </div>
    </div>
    </div>
    </div>
  </div>
</template>

<style scoped>
.table th {
  background-color: hsl(var(--b2) / 0.5);
}
</style>