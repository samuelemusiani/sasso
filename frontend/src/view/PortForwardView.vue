<template>
  <div class="min-h-screen ">
    <div class="container mx-auto px-4 py-8">
      <!-- Header -->
      <div class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 mb-6 shadow-xl liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-accent/5 ">
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-4">
            <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-lg">
              <Icon icon="material-symbols:router" class="text-2xl text-white" />
            </div>
            <div>
              <h1 class="text-3xl font-bold text-base-content">I Miei Port Forwarding</h1>
              <p class="text-base-content/70">Gestisci le tue richieste di port forwarding</p>
            </div>
          </div>
          
          <!-- Pulsante Nuova Richiesta -->
          <button @click="showCreateForm = !showCreateForm" 
                  class="btn btn-primary gap-2 shadow-lg hover:shadow-xl transition-all duration-300">
            <Icon icon="material-symbols:add" class="text-lg" />
            Nuova Richiesta
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
          <div class="stat-value text-2xl text-primary">{{ userStats.totalRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-success/10 to-success/20 border border-success/20 rounded-xl">
          <div class="stat-figure text-success">
            <Icon icon="material-symbols:check-circle" class="text-3xl" />
          </div>
          <div class="stat-title text-success/70">Approvati</div>
          <div class="stat-value text-2xl text-success">{{ userStats.approvedRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-warning/10 to-warning/20 border border-warning/20 rounded-xl">
          <div class="stat-figure text-warning">
            <Icon icon="material-symbols:schedule" class="text-3xl" />
          </div>
          <div class="stat-title text-warning/70">In attesa</div>
          <div class="stat-value text-2xl text-warning">{{ userStats.rejectedRequests }}</div>
        </div>
      </div>

      <!-- Form di Creazione -->
      <div v-if="showCreateForm" class="backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 mb-6 shadow-xl liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-accent/5 ">
        <h2 class="text-xl font-bold mb-4 flex items-center gap-2">
          <Icon icon="material-symbols:add-box" class="text-primary" />
          Nuova Richiesta Port Forwarding
        </h2>
        
        <form @submit.prevent="submitPortForwardRequest" class="space-y-4">
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
            <label class="label">
              <span class="label-text-alt text-base-content/60">Indirizzo IP del dispositivo di destinazione</span>
            </label>
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
            <label class="label">
              <span class="label-text-alt text-base-content/60">Porta del servizio sul dispositivo destinazione</span>
            </label>
          </div>
          
          <!-- Azioni -->
          <div class="flex gap-2 justify-end">
            <button type="button" 
                    @click="showCreateForm = false" 
                    class="btn btn-ghost">
              Annulla
            </button>
            <button type="submit" 
                    :disabled="isSubmitting"
                    class="btn btn-primary gap-2">
              <span v-if="isSubmitting" class="loading loading-spinner loading-sm"></span>
              <Icon v-else icon="material-symbols:send" />
              {{ isSubmitting ? 'Invio in corso...' : 'Invia Richiesta' }}
            </button>
          </div>
        </form>
      </div>

      <!-- Lista Port Forwards -->
      <div class=" backdrop-blur-sm border border-base-300/50 rounded-2xl shadow-xl overflow-hidden liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-accent/5 ">
        <div class="p-6 border-b border-base-300/50">
          <h2 class="text-xl font-bold flex items-center gap-2">
            <Icon icon="material-symbols:list" class="text-primary" />
            Le Tue Richieste
          </h2>
        </div>
        
        <div class="overflow-x-auto">
          <!-- Loading State -->
          <div v-if="isLoading" class="flex justify-center items-center h-64">
            <div class="loading loading-spinner loading-lg"></div>
            <span class="ml-4 text-lg">Caricamento richieste...</span>
          </div>
          
          <!-- Tabella port forwards -->
          <table v-else-if="portForwards.length > 0" class="table table-zebra w-full">
            <thead>
              <tr class="bg-base-200/50">
                <th class="font-semibold">ID</th>
                <th class="font-semibold">Porta Esterna</th>
                <th class="font-semibold">Destinazione</th>
                <th class="font-semibold">Stato</th>
                <th class="font-semibold">Creato il</th>
                <th class="font-semibold">Azioni</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="portForward in portForwards" :key="portForward.id" class="hover">
                <td>
                  <div class="font-mono text-sm font-medium">
                    {{ portForward.id }}
                  </div>
                </td>
                <td>
                  <div class="font-mono text-sm font-medium">
                    :{{ portForward.out_port }}
                  </div>
                </td>
                <td>
                  <div class="font-mono text-sm">
                    {{ portForward.dest_ip }}:{{ portForward.dest_port }}
                  </div>
                </td>
                <td>
                  <div class="badge gap-1" :class="getApprovalBadge(portForward.approved).class">
                    <Icon :icon="getApprovalBadge(portForward.approved).icon" class="text-xs" />
                    {{ getApprovalBadge(portForward.approved).text }}
                  </div>
                  <div v-if="portForward.status !== undefined" class="text-xs text-base-content/50">{{ portForward.status }}</div>
                </td>
                <td>
                  <div class="font-mono text-xs">
                    {{ portForward.created_at ? formatDate(String(portForward.created_at)) : '-' }}
                  </div>
                </td>
                <td>
                  <div class="flex gap-2">
                    <button @click="deletePortForward(portForward)"
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
            <p class="text-lg font-medium text-base-content/70 mb-2">Nessuna richiesta di port forwarding</p>
            <p class="text-base-content/50 mb-4">Crea la tua prima richiesta per iniziare</p>
            <button @click="showCreateForm = true" 
                    class="btn btn-primary gap-2">
              <Icon icon="material-symbols:add" />
              Nuova Richiesta
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'
import type { PortForward } from '@/types'

const portForwards = ref<PortForward[]>([])
const isLoading = ref(true)
const isSubmitting = ref(false)
const showCreateForm = ref(false)

// Form per nuova richiesta (struttura branch main)
const newPortForward = ref({
  dest_port: '',
  dest_ip: ''
})

// Statistiche delle richieste utente
const userStats = computed(() => ({
  totalRequests: portForwards.value.length,
  pendingRequests: portForwards.value.filter(pf => pf.status === 'pending').length,
  approvedRequests: portForwards.value.filter(pf => pf.status === 'approved').length,
  rejectedRequests: portForwards.value.filter(pf => pf.status === 'rejected').length,
  activeRequests: portForwards.value.filter(pf => pf.status === 'active').length
}))
// Variabile per filtro stato richieste
const selectedStatus = ref<string>('tutto')

// Funzione fetchUsers (placeholder)
function fetchUsers() {
  // Implementa qui la logica per recuperare gli utenti se serve
}
// Funzione di formattazione data
function formatDate(dateString: string) {
  const d = new Date(dateString)
  return d.toLocaleDateString('it-IT', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function fetchUserPortForwards() {
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

function submitPortForwardRequest() {
  if (!newPortForward.value.dest_port || !newPortForward.value.dest_ip) {
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

  isSubmitting.value = true
  api
    .post('/port-forwards', {
      dest_port: destPort,
      dest_ip: newPortForward.value.dest_ip,
    })
    .then(() => {
      fetchUserPortForwards()
      newPortForward.value = {
        dest_port: '',
        dest_ip: ''
      }
      showCreateForm.value = false
      globalNotifications.showSuccess('Richiesta di port forwarding inviata!')
    })
    .catch((err) => {
      console.error('Failed to add port forward:', err)
      globalNotifications.showError('Errore nell\'invio della richiesta di port forwarding')
    })
    .finally(() => {
      isSubmitting.value = false
    })
}

function deletePortForward(portForward: PortForward) {
  if (!confirm(`Sei sicuro di voler eliminare il port forwarding per ${portForward.dest_ip}:${portForward.dest_port}?`)) {
    return
  }

  api
    .delete(`/port-forwards/${portForward.id}`)
    .then(() => {
      fetchUserPortForwards()
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


onMounted(() => {
  fetchUserPortForwards()
})
</script>

<style scoped>
.table th {
  background-color: hsl(var(--b2) / 0.5);
}
</style>

<style scoped>
.liquid-glass-card {
  backdrop-filter: blur(20px);
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 16px;
  box-shadow: 
    0 25px 50px -12px rgba(0, 0, 0, 0.25),
    0 0 20px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}
</style>