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
  dest_port: '',
  name: '',
  user_id: '',
  target_ip: '',
  target_port: '',
  source_port: '',
  description: '',
})

// Variabili mancanti
const selectedStatus = ref<string>('all')
const users = ref<{ id: number; username: string; email: string }[]>([])

// Controlla se siamo in modalità creazione
const isCreateMode = computed(() => route.query.create === 'true')

// Port forwards filtrati per ricerca
const filteredPortForwards = computed(() => {
  let filtered = portForwards.value

  // Filtro per ricerca
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(
      (pf) => pf.dest_ip.includes(query) || pf.id.toString().includes(query),
    )
  }

  return filtered
})

// Statistiche dei port forwarding
const pfStats = computed(() => ({
  totalRequests: portForwards.value.length,
  approvedRequests: portForwards.value.filter((pf) => pf.approved).length,
  pendingRequests: portForwards.value.filter((pf) => !pf.approved).length,
  rejectedRequests: portForwards.value.filter((pf) => (pf.status ?? '') === 'rejected').length,
  activeRequests: portForwards.value.filter((pf) => (pf.status ?? '') === 'active').length,
}))

const statusOptions = [
  { value: 'all', label: 'Tutti', icon: 'material-symbols:list', color: 'text-base-content' },
  {
    value: 'pending',
    label: 'In attesa',
    icon: 'material-symbols:schedule',
    color: 'text-warning',
  },
  {
    value: 'approved',
    label: 'Approvati',
    icon: 'material-symbols:check-circle',
    color: 'text-success',
  },
  { value: 'active', label: 'Attivi', icon: 'material-symbols:play-circle', color: 'text-info' },
  { value: 'rejected', label: 'Rifiutati', icon: 'material-symbols:cancel', color: 'text-error' },
]

async function fetchPortForwards() {
  try {
    isLoading.value = true
    const res = await api.get('/admin/port-forwards')
    portForwards.value = res.data as PortForward[]
  } catch (error) {
    console.error('Errore nel caricamento dei port forwarding:', error)
    globalNotifications.showError('Errore nel caricamento dei port forwarding')
    portForwards.value = []
  } finally {
    isLoading.value = false
  }
}

async function createPortForward() {
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

  try {
    isProcessing.value = true
    await api.post('/port-forwards', {
      dest_ip: newPortForward.value.dest_ip,
      dest_port: destPort,
    })

    // Reset form
    newPortForward.value = {
      dest_ip: '',
      dest_port: '',
      name: '',
      user_id: '',
      target_ip: '',
      target_port: '',
      source_port: '',
      description: '',
    }

    showCreateModal.value = false
    await fetchPortForwards() // Ricarica la lista

    globalNotifications.showSuccess('Port forwarding creato con successo!')
  } catch (error) {
    console.error('Errore nella creazione del port forwarding:', error)
    globalNotifications.showError('Errore nella creazione del port forwarding')
  } finally {
    isProcessing.value = false
  }
}

// Nella struttura semplice, i port forward sono già approvati quando vengono creati
// Questa funzione ora è solo per eliminare se necessario

async function rejectPortForward(portForward: PortForward) {
  if (
    !confirm(
      `Sei sicuro di voler rifiutare il port forwarding "${portForward.name}" di ${portForward.user_name}?`,
    )
  ) {
    return
  }

  try {
    isProcessing.value = true
    await api.put(`/admin/port-forwards/${portForward.id}/reject`)

    // Aggiorna lo stato locale
    const index = portForwards.value.findIndex((pf) => pf.id === portForward.id)
    if (index !== -1) {
      portForwards.value[index].status = 'rejected'
    }

    globalNotifications.showSuccess(`Port forwarding "${portForward.name}" rifiutato`)
  } catch (error) {
    console.error('Errore nel rifiuto del port forwarding:', error)
    globalNotifications.showError('Errore nel rifiuto del port forwarding')
  } finally {
    isProcessing.value = false
  }
}

async function deletePortForward(portForward: PortForward) {
  if (
    !confirm(
      `Sei sicuro di voler eliminare definitivamente il port forwarding "${portForward.name}"?`,
    )
  ) {
    return
  }

  try {
    await api.delete(`/admin/port-forwards/${portForward.id}`)
    portForwards.value = portForwards.value.filter((pf) => pf.id !== portForward.id)
    globalNotifications.showSuccess('Port forwarding eliminato con successo!')
  } catch (error) {
    console.error("Errore nell'eliminazione del port forwarding:", error)
    globalNotifications.showError("Errore nell'eliminazione del port forwarding")
  }
}

function getStatusIcon(status: string): string {
  switch (status) {
    case 'pending':
      return 'material-symbols:schedule'
    case 'approved':
      return 'material-symbols:check-circle'
    case 'active':
      return 'material-symbols:play-circle'
    case 'rejected':
      return 'material-symbols:cancel'
    default:
      return 'material-symbols:help'
  }
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'pending':
      return 'text-warning'
    case 'approved':
      return 'text-success'
    case 'active':
      return 'text-info'
    case 'rejected':
      return 'text-error'
    default:
      return 'text-base-content'
  }
}

function getStatusBadgeClass(status: string): string {
  switch (status) {
    case 'pending':
      return 'badge-warning'
    case 'approved':
      return 'badge-success'
    case 'active':
      return 'badge-info'
    case 'rejected':
      return 'badge-error'
    default:
      return 'badge-ghost'
  }
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('it-IT', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function fetchUsers() {
  try {
    const res = await api.get('/admin/users')
    users.value = res.data
  } catch (error) {
    console.error('Errore nel caricamento degli utenti:', error)
    users.value = []
  }
}

async function approvePortForward(portForward: PortForward) {
  try {
    isProcessing.value = true
    await api.put(`/admin/port-forwards/${portForward.id}`, { approve: true })
    await fetchPortForwards()
    globalNotifications.showSuccess('Port forwarding approvato con successo!')
  } catch (error) {
    console.error("Errore nell'approvazione del port forwarding:", error)
    globalNotifications.showError("Errore nell'approvazione del port forwarding")
  } finally {
    isProcessing.value = false
  }
}

onMounted(async () => {
  await fetchPortForwards()

  // Se siamo in modalità creazione, carica gli utenti e apri il modal
  if (isCreateMode.value) {
    await fetchUsers()
    showCreateModal.value = true
  }
})
</script>

<template>
  <div class="min-h-screen">
    <div class="container mx-auto px-4 py-8">
      <!-- Header con effetto glass -->
      <div
        class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 mb-6 shadow-xl"
      >
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-4">
            <div
              class="w-12 h-12 rounded-xl bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center shadow-lg"
            >
              <IconifyIcon icon="material-symbols:router" class="text-2xl text-white" />
            </div>
            <div>
              <h1 class="text-3xl font-bold text-base-content">Approvazione Port Forwarding</h1>
              <p class="text-base-content/70">
                Gestisci le richieste di port forwarding degli utenti
              </p>
            </div>
          </div>

          <!-- Pulsante Crea Port Forward -->
          <button
            @click="
              showCreateModal = true
              fetchUsers()
            "
            class="btn btn-primary gap-2 shadow-lg hover:shadow-xl transition-all duration-300"
          >
            <IconifyIcon icon="material-symbols:add-box" class="text-lg" />
            Crea Port Forward
          </button>
        </div>
      </div>

      <!-- Statistiche -->
      <div class="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6 px-2">
        <div
          class="stat bg-gradient-to-br from-primary/10 to-primary/20 border border-primary/20 rounded-xl"
        >
          <div class="stat-figure text-primary">
            <IconifyIcon icon="material-symbols:list" class="text-3xl" />
          </div>
          <div class="stat-title text-primary/70">Totali</div>
          <div class="stat-value text-2xl text-primary">{{ pfStats.totalRequests }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-warning/10 to-warning/20 border border-warning/20 rounded-xl"
        >
          <div class="stat-figure text-warning">
            <IconifyIcon icon="material-symbols:schedule" class="text-3xl" />
          </div>
          <div class="stat-title text-warning/70">In attesa</div>
          <div class="stat-value text-2xl text-warning">{{ pfStats.pendingRequests }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-success/10 to-success/20 border border-success/20 rounded-xl"
        >
          <div class="stat-figure text-success">
            <IconifyIcon icon="material-symbols:check-circle" class="text-3xl" />
          </div>
          <div class="stat-title text-success/70">Approvati</div>
          <div class="stat-value text-2xl text-success">{{ pfStats.approvedRequests }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-info/10 to-info/20 border border-info/20 rounded-xl"
        >
          <div class="stat-figure text-info">
            <IconifyIcon icon="material-symbols:play-circle" class="text-3xl" />
          </div>
          <div class="stat-title text-info/70">Attivi</div>
          <div class="stat-value text-2xl text-info">{{ pfStats.activeRequests }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-error/10 to-error/20 border border-error/20 rounded-xl"
        >
          <div class="stat-figure text-error">
            <IconifyIcon icon="material-symbols:cancel" class="text-3xl" />
          </div>
          <div class="stat-title text-error/70">Rifiutati</div>
          <div class="stat-value text-2xl text-error">{{ pfStats.rejectedRequests }}</div>
        </div>
      </div>

      <!-- Controlli principali -->
      <div class="flex flex-col md:flex-row md:items-center gap-4 mb-6 px-2">
        <!-- Barra di ricerca -->
        <div class="flex items-center gap-3 flex-1">
          <IconifyIcon icon="material-symbols:search" class="text-base-content/60 text-xl" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Cerca per nome, utente, IP..."
            class="input input-bordered flex-1"
          />
        </div>

        <!-- Filtro stato -->
        <div class="flex gap-2 shrink-0">
          <select v-model="selectedStatus" class="select select-bordered">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>
      </div>

      <!-- Lista port forwarding -->
      <div class="px-2">
        <div
          class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl shadow-lg overflow-hidden"
        >
          <!-- Loading state -->
          <div v-if="isLoading" class="p-8 text-center">
            <span class="loading loading-spinner loading-lg text-primary"></span>
            <p class="mt-2 text-base-content/70">Caricamento richieste port forwarding...</p>
          </div>

          <!-- Tabella port forwarding -->
          <div v-else-if="filteredPortForwards.length > 0" class="overflow-x-auto">
            <table class="table table-zebra w-full">
              <thead>
                <tr class="border-base-300">
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:tag" class="text-sm" />
                      ID
                    </div>
                  </th>
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:person" class="text-sm" />
                      Utente
                    </div>
                  </th>
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:router" class="text-sm" />
                      Port Forwarding
                    </div>
                  </th>
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:network-node" class="text-sm" />
                      Routing
                    </div>
                  </th>
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:flag" class="text-sm" />
                      Stato
                    </div>
                  </th>
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:schedule" class="text-sm" />
                      Data
                    </div>
                  </th>
                  <th class="bg-base-200/50 text-center">Azioni</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="pf in filteredPortForwards" :key="pf.id" class="hover">
                  <td class="font-mono text-sm">{{ pf.id }}</td>
                  <td>
                    <div class="flex items-center gap-3">
                      <div
                        class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center"
                      >
                        <IconifyIcon icon="material-symbols:person" class="text-primary text-xl" />
                      </div>
                      <div>
                        <div class="font-bold">{{ pf.user_name }}</div>
                        <div class="text-sm text-base-content/70">ID: {{ pf.user_id }}</div>
                      </div>
                    </div>
                  </td>
                  <td>
                    <div>
                      <div class="font-bold">{{ pf.name }}</div>
                      <div class="text-sm text-base-content/70">{{ pf.description }}</div>
                    </div>
                  </td>
                  <td class="font-mono text-sm">
                    <div class="flex items-center gap-2">
                      <span class="badge badge-outline">{{ pf.source_port }}</span>
                      <IconifyIcon
                        icon="material-symbols:arrow-forward"
                        class="text-base-content/60"
                      />
                      <span class="text-primary">{{ pf.target_ip }}:{{ pf.target_port }}</span>
                    </div>
                  </td>
                  <td>
                    <div class="flex items-center gap-2">
                      <Icon
                        :icon="getStatusIcon(pf.status ?? 'unknown')"
                        :class="getStatusColor(pf.status ?? 'unknown')"
                      />
                      <span
                        class="badge badge-sm"
                        :class="getStatusBadgeClass(pf.status ?? 'unknown')"
                      >
                        {{ (pf.status ?? 'unknown').toUpperCase() }}
                      </span>
                    </div>
                  </td>
                  <td class="text-sm text-base-content/70">
                    {{ formatDate(pf.created_at ?? '') }}
                  </td>
                  <td>
                    <div class="flex gap-1 justify-center">
                      <!-- Pulsante approva -->
                      <button
                        v-if="pf.status === 'pending'"
                        @click="approvePortForward(pf)"
                        class="btn btn-ghost btn-sm gap-1 hover:btn-success"
                        :disabled="isProcessing"
                      >
                        <IconifyIcon icon="material-symbols:check" />
                        Approva
                      </button>

                      <!-- Pulsante rifiuta -->
                      <button
                        v-if="pf.status === 'pending'"
                        @click="rejectPortForward(pf)"
                        class="btn btn-ghost btn-sm gap-1 hover:btn-error"
                        :disabled="isProcessing"
                      >
                        <IconifyIcon icon="material-symbols:close" />
                        Rifiuta
                      </button>

                      <!-- Pulsante elimina -->
                      <button
                        @click="deletePortForward(pf)"
                        class="btn btn-ghost btn-sm gap-1 hover:btn-error"
                        :disabled="isProcessing"
                      >
                        <IconifyIcon icon="material-symbols:delete" />
                        Elimina
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Stato vuoto -->
          <div v-else-if="portForwards.length === 0" class="p-8 text-center">
            <Icon
              icon="material-symbols:router-outline"
              class="text-6xl text-base-content/30 mb-4"
            />
            <p class="text-lg font-medium text-base-content/70 mb-2">
              Nessuna richiesta di port forwarding
            </p>
            <p class="text-base-content/50">Le richieste degli utenti appariranno qui</p>
          </div>

          <!-- Stato vuoto per ricerca -->
          <div v-else class="p-8 text-center">
            <IconifyIcon
              icon="material-symbols:search-off"
              class="text-6xl text-base-content/30 mb-4"
            />
            <p class="text-base-content/70">Nessuna richiesta trovata per i criteri selezionati</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Creazione Port Forward -->
    <div v-if="showCreateModal" class="modal modal-open">
      <div class="modal-box w-11/12 max-w-2xl bg-base-100 border border-base-300">
        <h3 class="font-bold text-lg mb-6 flex items-center gap-2">
          <IconifyIcon icon="material-symbols:add-box" class="text-orange-500" />
          Crea Nuovo Port Forward (Pre-approvato)
        </h3>

        <div class="alert alert-info mb-4">
          <IconifyIcon icon="material-symbols:info" />
          <span class="text-sm"
            >I port forward creati dagli admin vengono automaticamente approvati e attivati.</span
          >
        </div>

        <form @submit.prevent="createPortForward" class="space-y-4">
          <!-- Nome -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Nome *</span>
            </label>
            <input
              v-model="newPortForward.name"
              type="text"
              placeholder="Nome identificativo del port forward"
              class="input input-bordered w-full"
              required
            />
          </div>

          <!-- Utente -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Utente *</span>
            </label>
            <select v-model="newPortForward.user_id" class="select select-bordered w-full" required>
              <option value="">Seleziona utente</option>
              <option v-for="user in users" :key="user.id" :value="user.id">
                {{ user.username }} ({{ user.email }})
              </option>
            </select>
          </div>

          <!-- IP Target e Porta Target -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text font-medium">IP Target *</span>
              </label>
              <input
                v-model="newPortForward.target_ip"
                type="text"
                placeholder="192.168.1.100"
                class="input input-bordered w-full"
                pattern="^(\d{1,3}\.){3}\d{1,3}$"
                required
              />
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text font-medium">Porta Target *</span>
              </label>
              <input
                v-model="newPortForward.target_port"
                type="number"
                placeholder="8080"
                class="input input-bordered w-full"
                min="1"
                max="65535"
                required
              />
            </div>
          </div>

          <!-- Porta Sorgente -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Porta Sorgente *</span>
            </label>
            <input
              v-model="newPortForward.source_port"
              type="number"
              placeholder="8080"
              class="input input-bordered w-full"
              min="1"
              max="65535"
              required
            />
          </div>

          <!-- Descrizione -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-medium">Descrizione</span>
            </label>
            <textarea
              v-model="newPortForward.description"
              placeholder="Descrizione opzionale del port forward"
              class="textarea textarea-bordered w-full h-20"
            ></textarea>
          </div>

          <!-- Azioni -->
          <div class="modal-action">
            <button type="button" @click="showCreateModal = false" class="btn btn-ghost">
              Annulla
            </button>
            <button type="submit" :disabled="isProcessing" class="btn btn-primary gap-2">
              <span v-if="isProcessing" class="loading loading-spinner loading-sm"></span>
              <IconifyIcon v-else icon="material-symbols:add-box" />
              {{ isProcessing ? 'Creazione...' : 'Crea Port Forward' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
