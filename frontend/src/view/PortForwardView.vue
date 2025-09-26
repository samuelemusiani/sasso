<template>
  <div class="min-h-screen bg-gradient-to-br from-base-100 to-base-200">
    <div class="container mx-auto px-4 py-8">
      <!-- Header -->
      <div class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 mb-6 shadow-xl">
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
      <div class="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6 px-2">
        <div class="stat bg-gradient-to-br from-primary/10 to-primary/20 border border-primary/20 rounded-xl">
          <div class="stat-figure text-primary">
            <Icon icon="material-symbols:list" class="text-3xl" />
          </div>
          <div class="stat-title text-primary/70">Totali</div>
          <div class="stat-value text-2xl text-primary">{{ userStats.totalRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-warning/10 to-warning/20 border border-warning/20 rounded-xl">
          <div class="stat-figure text-warning">
            <Icon icon="material-symbols:schedule" class="text-3xl" />
          </div>
          <div class="stat-title text-warning/70">In attesa</div>
          <div class="stat-value text-2xl text-warning">{{ userStats.pendingRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-success/10 to-success/20 border border-success/20 rounded-xl">
          <div class="stat-figure text-success">
            <Icon icon="material-symbols:check-circle" class="text-3xl" />
          </div>
          <div class="stat-title text-success/70">Approvati</div>
          <div class="stat-value text-2xl text-success">{{ userStats.approvedRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-info/10 to-info/20 border border-info/20 rounded-xl">
          <div class="stat-figure text-info">
            <Icon icon="material-symbols:play-circle" class="text-3xl" />
          </div>
          <div class="stat-title text-info/70">Attivi</div>
          <div class="stat-value text-2xl text-info">{{ userStats.activeRequests }}</div>
        </div>
        
        <div class="stat bg-gradient-to-br from-error/10 to-error/20 border border-error/20 rounded-xl">
          <div class="stat-figure text-error">
            <Icon icon="material-symbols:cancel" class="text-3xl" />
          </div>
          <div class="stat-title text-error/70">Rifiutati</div>
          <div class="stat-value text-2xl text-error">{{ userStats.rejectedRequests }}</div>
        </div>
      </div>

      <!-- Form di Creazione -->
      <div v-if="showCreateForm" class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 mb-6 shadow-xl">
        <h2 class="text-xl font-bold mb-4 flex items-center gap-2">
          <Icon icon="material-symbols:add-box" class="text-primary" />
          Nuova Richiesta Port Forwarding
        </h2>
        
        <form @submit.prevent="submitPortForwardRequest" class="space-y-4">
          <!-- Nome -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Nome Identificativo *</span>
            </label>
            <input v-model="newPortForward.name" 
                   type="text" 
                   placeholder="es. Server Web Personale, Database MySQL, ecc." 
                   class="input input-bordered w-full"
                   required />
            <label class="label">
              <span class="label-text-alt text-base-content/60">Un nome che ti aiuti a riconoscere questo port forwarding</span>
            </label>
          </div>
          
          <!-- IP e Porta Target -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text font-medium">IP Target *</span>
              </label>
              <input v-model="newPortForward.target_ip" 
                     type="text" 
                     placeholder="192.168.1.100" 
                     class="input input-bordered w-full"
                     pattern="^(\d{1,3}\.){3}\d{1,3}$"
                     required />
              <label class="label">
                <span class="label-text-alt text-base-content/60">Indirizzo IP del dispositivo di destinazione</span>
              </label>
            </div>
            
            <div class="form-control">
              <label class="label">
                <span class="label-text font-medium">Porta Target *</span>
              </label>
              <input v-model="newPortForward.target_port" 
                     type="number" 
                     placeholder="8080" 
                     class="input input-bordered w-full"
                     min="1" max="65535"
                     required />
              <label class="label">
                <span class="label-text-alt text-base-content/60">Porta del servizio sul dispositivo target</span>
              </label>
            </div>
          </div>
          
          <!-- Porta Sorgente -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Porta Sorgente *</span>
            </label>
            <input v-model="newPortForward.source_port" 
                   type="number" 
                   placeholder="8080" 
                   class="input input-bordered w-full"
                   min="1" max="65535"
                   required />
            <label class="label">
              <span class="label-text-alt text-base-content/60">Porta attraverso cui il servizio sarà accessibile dall'esterno</span>
            </label>
          </div>
          
          <!-- Descrizione -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Descrizione</span>
            </label>
            <textarea v-model="newPortForward.description" 
                      placeholder="Descrizione dettagliata del servizio e del motivo della richiesta..."
                      class="textarea textarea-bordered w-full h-24"></textarea>
            <label class="label">
              <span class="label-text-alt text-base-content/60">Spiega brevemente perché hai bisogno di questo port forwarding</span>
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
      <div class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl shadow-xl overflow-hidden">
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
                <th class="font-semibold">Nome</th>
                <th class="font-semibold">Target</th>
                <th class="font-semibold">Porta Sorgente</th>
                <th class="font-semibold">Stato</th>
                <th class="font-semibold">Data Richiesta</th>
                <th class="font-semibold">Azioni</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="portForward in portForwards" :key="portForward.id" class="hover">
                <td>
                  <div>
                    <div class="font-medium">{{ portForward.name }}</div>
                    <div v-if="portForward.description" class="text-sm text-base-content/60 truncate max-w-xs" :title="portForward.description">
                      {{ portForward.description }}
                    </div>
                  </div>
                </td>
                <td>
                  <div class="font-mono text-sm">
                    {{ portForward.target_ip }}:{{ portForward.target_port }}
                  </div>
                </td>
                <td>
                  <div class="font-mono text-sm font-medium">
                    :{{ portForward.source_port }}
                  </div>
                </td>
                <td>
                  <div class="badge gap-1" :class="getStatusBadge(portForward.status).class">
                    <Icon :icon="getStatusBadge(portForward.status).icon" class="text-xs" />
                    {{ getStatusBadge(portForward.status).text }}
                  </div>
                </td>
                <td>
                  <div class="text-sm">
                    {{ new Date(portForward.created_at).toLocaleDateString('it-IT') }}
                  </div>
                </td>
                <td>
                  <div class="flex gap-2">
                    <!-- Elimina solo se pending o rejected -->
                    <button v-if="['pending', 'rejected'].includes(portForward.status)"
                            @click="deletePortForward(portForward)"
                            class="btn btn-ghost btn-xs text-error hover:bg-error/10">
                      <Icon icon="material-symbols:delete" class="text-sm" />
                    </button>
                    <span v-else class="text-xs text-base-content/50">
                      {{ portForward.status === 'approved' ? 'Approvato' : 'Attivo' }}
                    </span>
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

// Form per nuova richiesta
const newPortForward = ref({
  name: '',
  target_ip: '',
  target_port: '',
  source_port: '',
  description: ''
})

// Statistiche delle richieste utente
const userStats = computed(() => ({
  totalRequests: portForwards.value.length,
  pendingRequests: portForwards.value.filter(pf => pf.status === 'pending').length,
  approvedRequests: portForwards.value.filter(pf => pf.status === 'approved').length,
  activeRequests: portForwards.value.filter(pf => pf.status === 'active').length,
  rejectedRequests: portForwards.value.filter(pf => pf.status === 'rejected').length
}))

async function fetchUserPortForwards() {
  try {
    isLoading.value = true
    const res = await api.get('/user/port-forwards')
    portForwards.value = res.data as PortForward[]
  } catch (error) {
    console.error('Errore nel caricamento dei port forwarding:', error)
    globalNotifications.showError('Errore nel caricamento dei port forwarding')
    portForwards.value = []
  } finally {
    isLoading.value = false
  }
}

async function submitPortForwardRequest() {
  if (!newPortForward.value.name || !newPortForward.value.target_ip || 
      !newPortForward.value.target_port || !newPortForward.value.source_port) {
    globalNotifications.showError('Compila tutti i campi obbligatori')
    return
  }

  // Validazione IP
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/
  if (!ipRegex.test(newPortForward.value.target_ip)) {
    globalNotifications.showError('Inserisci un indirizzo IP valido')
    return
  }

  // Validazione porte
  const targetPort = parseInt(newPortForward.value.target_port)
  const sourcePort = parseInt(newPortForward.value.source_port)
  
  if (targetPort < 1 || targetPort > 65535 || sourcePort < 1 || sourcePort > 65535) {
    globalNotifications.showError('Le porte devono essere comprese tra 1 e 65535')
    return
  }

  try {
    isSubmitting.value = true
    await api.post('/user/port-forwards', {
      name: newPortForward.value.name,
      target_ip: newPortForward.value.target_ip,
      target_port: targetPort,
      source_port: sourcePort,
      description: newPortForward.value.description
    })
    
    // Reset form
    newPortForward.value = {
      name: '',
      target_ip: '',
      target_port: '',
      source_port: '',
      description: ''
    }
    
    showCreateForm.value = false
    await fetchUserPortForwards() // Ricarica la lista
    
    globalNotifications.showSuccess('Richiesta di port forwarding inviata! Sarà esaminata dagli amministratori.')
  } catch (error) {
    console.error('Errore nell\'invio della richiesta:', error)
    globalNotifications.showError('Errore nell\'invio della richiesta di port forwarding')
  } finally {
    isSubmitting.value = false
  }
}

async function deletePortForward(portForward: PortForward) {
  if (!confirm(`Sei sicuro di voler eliminare il port forwarding "${portForward.name}"?`)) {
    return
  }

  try {
    await api.delete(`/user/port-forwards/${portForward.id}`)
    await fetchUserPortForwards() // Ricarica la lista
    globalNotifications.showSuccess('Port forwarding eliminato con successo')
  } catch (error) {
    console.error('Errore nell\'eliminazione del port forwarding:', error)
    globalNotifications.showError('Errore nell\'eliminazione del port forwarding')
  }
}

function getStatusBadge(status: string) {
  switch (status) {
    case 'pending':
      return { class: 'badge-warning', icon: 'material-symbols:schedule', text: 'In attesa' }
    case 'approved':
      return { class: 'badge-success', icon: 'material-symbols:check-circle', text: 'Approvato' }
    case 'active':
      return { class: 'badge-info', icon: 'material-symbols:play-circle', text: 'Attivo' }
    case 'rejected':
      return { class: 'badge-error', icon: 'material-symbols:cancel', text: 'Rifiutato' }
    default:
      return { class: 'badge-ghost', icon: 'material-symbols:help', text: 'Sconosciuto' }
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