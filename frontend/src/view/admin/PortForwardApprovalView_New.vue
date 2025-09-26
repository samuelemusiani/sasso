<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'
import type { PortForward } from '@/types'

const route = useRoute()
const portForwards = ref<PortForward[]>([])
const isLoading = ref(true)
const searchQuery = ref('')
const isProcessing = ref(false)

// Modalità creazione
const showCreateModal = ref(false)
const newPortForward = ref({
  dest_ip: '',
  dest_port: ''
})

// Controlla se siamo in modalità creazione
const isCreateMode = computed(() => route.query.create === 'true')

// Port forwards filtrati per ricerca
const filteredPortForwards = computed(() => {
  let filtered = portForwards.value
  
  // Filtro per ricerca
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(pf => 
      pf.dest_ip.includes(query) ||
      pf.id.toString().includes(query)
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
    .get('/port-forwards')
    .then((res) => {
      portForwards.value = res.data as PortForward[]
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
  api
    .post('/port-forwards', {
      dest_ip: newPortForward.value.dest_ip,
      dest_port: destPort
    })
    .then(() => {
      fetchPortForwards()
      newPortForward.value = {
        dest_ip: '',
        dest_port: ''
      }
      showCreateModal.value = false
      globalNotifications.showSuccess('Port forwarding creato con successo!')
    })
    .catch((err) => {
      console.error('Failed to add port forward:', err)
      globalNotifications.showError('Errore nella creazione del port forwarding')
    })
    .finally(() => {
      isProcessing.value = false
    })
}

function deletePortForward(portForward: PortForward) {
  if (!confirm(`Sei sicuro di voler eliminare il port forwarding ${portForward.dest_ip}:${portForward.dest_port}?`)) {
    return
  }

  api
    .delete(`/port-forwards/${portForward.id}`)
    .then(() => {
      fetchPortForwards()
      globalNotifications.showSuccess('Port forwarding eliminato con successo')
    })
    .catch((err) => {
      console.error('Failed to delete Port Forward:', err)
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

onMounted(async () => {
  await fetchPortForwards()
  
  // Se siamo in modalità creazione, apri il modal
  if (isCreateMode.value) {
    showCreateModal.value = true
  }
})
</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-base-100 to-base-200">
    <div class="container mx-auto px-4 py-8">
      <!-- Header con effetto glass -->
      <div class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 mb-6 shadow-xl">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-4">
            <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center shadow-lg">
              <Icon icon="material-symbols:router" class="text-2xl text-white" />
            </div>
            <div>
              <h1 class="text-3xl font-bold text-base-content">Gestione Port Forwarding</h1>
              <p class="text-base-content/70">Visualizza e gestisci tutti i port forwarding del sistema</p>
            </div>
          </div>
          
          <!-- Pulsante Crea Port Forward -->
          <button @click="showCreateModal = true" 
                  class="btn btn-primary gap-2 shadow-lg hover:shadow-xl transition-all duration-300">
            <Icon icon="material-symbols:add-box" class="text-lg" />
            Crea Port Forward
          </button>
        </div>
      </div>

      <!-- Statistiche -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6 px-2">
        <div class="stat bg-gradient-to-br from-primary/10 to-primary/20 border border-primary/20 rounded-xl">
          <div class="stat-figure text-primary">
            <Icon icon="material-symbols:list" class="text-3xl" />
          </div>
          <div class="stat-title text-primary/70">Totali</div>
          <div class="stat-value text-2xl text-primary">{{ pfStats.totalRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-success/10 to-success/20 border border-success/20 rounded-xl">
          <div class="stat-figure text-success">
            <Icon icon="material-symbols:check-circle" class="text-3xl" />
          </div>
          <div class="stat-title text-success/70">Approvati</div>
          <div class="stat-value text-2xl text-success">{{ pfStats.approvedRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-warning/10 to-warning/20 border border-warning/20 rounded-xl">
          <div class="stat-figure text-warning">
            <Icon icon="material-symbols:schedule" class="text-3xl" />
          </div>
          <div class="stat-title text-warning/70">In Attesa</div>
          <div class="stat-value text-2xl text-warning">{{ pfStats.pendingRequests }}</div>
        </div>
      </div>

      <!-- Controlli di Ricerca -->
      <div class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-4 mb-6 shadow-xl">
        <div class="flex gap-4 items-center">
          <div class="form-control flex-1">
            <input v-model="searchQuery" 
                   type="text" 
                   placeholder="Cerca per IP o ID..." 
                   class="input input-bordered w-full" />
          </div>
        </div>
      </div>

      <!-- Lista Port Forwards -->
      <div class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl shadow-xl overflow-hidden">
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
                    <button @click="deletePortForward(pf)"
                            class="btn btn-ghost btn-xs text-error hover:bg-error/10">
                      <Icon icon="material-symbols:delete" class="text-sm" />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          
          <!-- Stato vuoto -->
          <div v-else class="p-8 text-center">
            <Icon icon="material-symbols:router-outline" class="text-6xl text-base-content/30 mb-4" />
            <p class="text-lg font-medium text-base-content/70 mb-2">Nessun port forwarding</p>
            <p class="text-base-content/50 mb-4">I port forwarding appariranno qui</p>
            <button @click="showCreateModal = true" 
                    class="btn btn-primary gap-2">
              <Icon icon="material-symbols:add" />
              Crea Port Forward
            </button>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Modal Creazione Port Forward -->
    <div v-if="showCreateModal" class="modal modal-open">
      <div class="modal-box w-11/12 max-w-2xl bg-base-100 border border-base-300">
        <h3 class="font-bold text-lg mb-6 flex items-center gap-2">
          <Icon icon="material-symbols:add-box" class="text-orange-500" />
          Crea Nuovo Port Forward
        </h3>
        
        <form @submit.prevent="createPortForward" class="space-y-4">
          <!-- IP Destinazione -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">IP Destinazione *</span>
            </label>
            <input v-model="newPortForward.dest_ip" 
                   type="text" 
                   placeholder="192.168.1.100" 
                   class="input input-bordered w-full"
                   pattern="^(\d{1,3}\.){3}\d{1,3}$"
                   required />
          </div>
          
          <!-- Porta Destinazione -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Porta Destinazione *</span>
            </label>
            <input v-model="newPortForward.dest_port" 
                   type="number" 
                   placeholder="8080" 
                   class="input input-bordered w-full"
                   min="1" max="65535"
                   required />
          </div>
          
          <!-- Azioni -->
          <div class="modal-action">
            <button type="button" 
                    @click="showCreateModal = false" 
                    class="btn btn-ghost">
              Annulla
            </button>
            <button type="submit" 
                    :disabled="isProcessing"
                    class="btn btn-primary gap-2">
              <span v-if="isProcessing" class="loading loading-spinner loading-sm"></span>
              <Icon v-else icon="material-symbols:add-box" />
              {{ isProcessing ? 'Creazione...' : 'Crea Port Forward' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.table th {
  background-color: hsl(var(--b2) / 0.5);
}
</style>