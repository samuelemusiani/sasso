<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/lib/api'
import type { User, VM, Net } from '@/types'

const user = ref<User | null>(null)
const vms = ref<VM[]>([])
const nets = ref<Net[]>([])
const isLoading = ref(true)

// Dati simulati per i log (in futuro da API)
const logs = ref([
  {
    id: 1,
    timestamp: '2025-09-25 18:45:23',
    type: 'info',
    message: 'VM #1001 avviata con successo',
    user: 'admin',
  },
  {
    id: 2,
    timestamp: '2025-09-25 18:44:15',
    type: 'warning',
    message: 'Utilizzo CPU al 85% per VM #1002',
    user: 'user1',
  },
  {
    id: 3,
    timestamp: '2025-09-25 18:43:01',
    type: 'error',
    message: 'Errore nella creazione della rete "test-net"',
    user: 'user2',
  },
  {
    id: 4,
    timestamp: '2025-09-25 18:42:33',
    type: 'info',
    message: 'Nuova SSH key aggiunta per utente "dev-team"',
    user: 'admin',
  },
  {
    id: 5,
    timestamp: '2025-09-25 18:41:45',
    type: 'success',
    message: 'Backup VM #1003 completato',
    user: 'system',
  },
  {
    id: 6,
    timestamp: '2025-09-25 18:40:12',
    type: 'warning',
    message: 'Spazio disco insufficiente per VM #1004',
    user: 'user3',
  },
  {
    id: 7,
    timestamp: '2025-09-25 18:39:58',
    type: 'info',
    message: 'Aggiornamento configurazione rete completato',
    user: 'admin',
  },
  {
    id: 8,
    timestamp: '2025-09-25 18:38:22',
    type: 'error',
    message: 'Connessione VPN interrotta per utente "mobile-user"',
    user: 'system',
  },
])

// Funzioni per recuperare i dati
async function fetchUserData() {
  try {
    const response = await api.get('/whoami')
    user.value = response.data as User
  } catch (error) {
    console.error('Errore nel recuperare i dati utente:', error)
  }
}

async function fetchVMs() {
  try {
    const response = await api.get('/vm')
    vms.value = response.data as VM[]
  } catch (error) {
    console.error('Errore nel recuperare le VM:', error)
  }
}

async function fetchNets() {
  try {
    const response = await api.get('/net')
    nets.value = response.data as Net[]
  } catch (error) {
    console.error('Errore nel recuperare le reti:', error)
  }
}

// Costanti per il cerchio di progresso
const CIRCLE_RADIUS = 35
const CIRCLE_CIRCUMFERENCE = 2 * Math.PI * CIRCLE_RADIUS

// Calcolo delle metriche in tempo reale
const metrics = computed(() => {
  if (!user.value) return []

  // Calcola risorse utilizzate dalle VM
  const usedCores = vms.value.reduce((sum, vm) => sum + vm.cores, 0)
  const usedRam = vms.value.reduce((sum, vm) => sum + vm.ram, 0)
  const usedDisk = vms.value.reduce((sum, vm) => sum + vm.disk, 0)
  const usedNets = nets.value.length

  // Calcola le percentuali con maggiore precisione
  const cpuPercentage =
    user.value.max_cores > 0
      ? Math.min(100, Math.round((usedCores / user.value.max_cores) * 100))
      : 0
  const ramPercentage =
    user.value.max_ram > 0 ? Math.min(100, Math.round((usedRam / user.value.max_ram) * 100)) : 0
  const diskPercentage =
    user.value.max_disk > 0 ? Math.min(100, Math.round((usedDisk / user.value.max_disk) * 100)) : 0
  const netPercentage =
    user.value.max_nets > 0 ? Math.min(100, Math.round((usedNets / user.value.max_nets) * 100)) : 0

  return [
    {
      title: 'CPU',
      percentage: cpuPercentage,
      icon: 'material-symbols:memory',
      used: usedCores,
      total: user.value.max_cores,
      unit: 'cores',
      strokeOffset: CIRCLE_CIRCUMFERENCE - (cpuPercentage / 100) * CIRCLE_CIRCUMFERENCE,
    },
    {
      title: 'RAM',
      percentage: ramPercentage,
      icon: 'material-symbols:storage',
      used: usedRam,
      total: user.value.max_ram,
      unit: 'MB',
      strokeOffset: CIRCLE_CIRCUMFERENCE - (ramPercentage / 100) * CIRCLE_CIRCUMFERENCE,
    },
    {
      title: 'Disco',
      percentage: diskPercentage,
      icon: 'material-symbols:hard-drive',
      used: usedDisk,
      total: user.value.max_disk,
      unit: 'GB',
      strokeOffset: CIRCLE_CIRCUMFERENCE - (diskPercentage / 100) * CIRCLE_CIRCUMFERENCE,
    },
    {
      title: 'Rete',
      percentage: netPercentage,
      icon: 'material-symbols:network-wifi',
      used: usedNets,
      total: user.value.max_nets,
      unit: 'reti',
      strokeOffset: CIRCLE_CIRCUMFERENCE - (netPercentage / 100) * CIRCLE_CIRCUMFERENCE,
    },
  ]
})

// Aggiorna i dati all'avvio e ogni 10 secondi
onMounted(async () => {
  await Promise.all([fetchUserData(), fetchVMs(), fetchNets()])
  isLoading.value = false

  // Aggiornamento periodico ogni 10 secondi
  const interval = setInterval(async () => {
    await Promise.all([fetchVMs(), fetchNets()])

    // Simula l'aggiunta di nuovi log (in futuro da API)
    if (Math.random() > 0.7) {
      const newLog = {
        id: Date.now(),
        timestamp: new Date().toLocaleString('it-IT'),
        type: ['info', 'warning', 'success', 'error'][Math.floor(Math.random() * 4)],
        message: [
          'Sistema monitorato correttamente',
          'Nuova VM creata automaticamente',
          'Backup completato',
          'Connessione ristabilita',
          'Configurazione aggiornata',
        ][Math.floor(Math.random() * 5)],
        user: ['system', 'admin', 'user'][Math.floor(Math.random() * 3)],
      }
      logs.value.unshift(newLog)

      // Mantieni solo gli ultimi 20 log
      if (logs.value.length > 20) {
        logs.value = logs.value.slice(0, 20)
      }
    }
  }, 10000)

  // Cleanup quando il componente viene distrutto
  return () => clearInterval(interval)
})
</script>

<template>
  <!-- Contenuto principale della dashboard -->
  <div class="h-full overflow-auto">
    <!-- Header -->
    <div class="mb-8 px-2">
      <h1 class="text-3xl font-bold text-base-content mb-2">Dashboard Risorse</h1>
      <p class="text-base-content/70" v-if="user">
        Quota utilizzata da <span class="font-semibold">{{ user.username }}</span>
      </p>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center items-center h-64">
      <div class="loading loading-spinner loading-lg"></div>
      <span class="ml-4 text-lg">Caricamento dati...</span>
    </div>

    <!-- Griglia delle metriche con effetto liquid glass intensificato -->
    <div
      v-else
      class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 mb-8 px-2"
    >
      <div
        v-for="metric in metrics"
        :key="metric.title"
        class="card shadow-2xl border border-white/30 bg-gradient-to-br from-white/15 to-white/8 backdrop-blur-2xl backdrop-saturate-200 hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25),0_0_20px_rgba(255,255,255,0.1)] hover:scale-[1.02] hover:border-white/40 transition-all duration-300 ease-out hover:bg-gradient-to-br hover:from-white/20 hover:to-white/12 overflow-hidden relative before:absolute before:inset-0 before:bg-gradient-to-r before:from-transparent before:via-white/5 before:to-transparent before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300"
      >
        <div class="card-body p-6">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 w-40">
              <div class="btn btn-square btn-md rounded-xl btn-primary p-0 m-1 flex-shrink-0">
                <IconifyIcon :icon="metric.icon" class="text-xl" />
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="font-bold text-lg text-base-content truncate drop-shadow-sm mb-1">
                  {{ metric.title }}
                </h3>
                <span
                  class="text-sm font-semibold text-base-content/80 drop-shadow-sm truncate block"
                >
                  {{ metric.used }} / {{ metric.total }} {{ metric.unit }}
                </span>
              </div>
            </div>

            <!-- Indicatore circolare di percentuale perfezionato -->
            <div class="relative flex-shrink-0">
              <svg class="w-16 h-16 transform -rotate-90" viewBox="0 0 100 100">
                <!-- Cerchio di sfondo con glow sottile -->
                <circle
                  cx="50"
                  cy="50"
                  :r="CIRCLE_RADIUS"
                  stroke="currentColor"
                  stroke-width="6"
                  fill="none"
                  class="text-base-300/50 drop-shadow-sm"
                />
                <!-- Cerchio di progresso con glow controllato -->
                <circle
                  cx="50"
                  cy="50"
                  :r="CIRCLE_RADIUS"
                  stroke="currentColor"
                  stroke-width="6"
                  fill="none"
                  :stroke-dasharray="CIRCLE_CIRCUMFERENCE"
                  :stroke-dashoffset="metric.strokeOffset"
                  stroke-linecap="round"
                  :class="{
                    'text-success': metric.percentage < 60,
                    'text-warning': metric.percentage >= 60 && metric.percentage < 80,
                    'text-error': metric.percentage >= 80,
                  }"
                  class="transition-all duration-1000 ease-out"
                  :style="{
                    filter: `drop-shadow(0 0 4px ${
                      metric.percentage < 60
                        ? 'rgb(34 197 94 / 0.6)'
                        : metric.percentage < 80
                          ? 'rgb(245 158 11 / 0.6)'
                          : 'rgb(239 68 68 / 0.6)'
                    })`,
                  }"
                />
              </svg>
              <!-- Percentuale al centro con glow -->
              <div class="absolute inset-0 flex items-center justify-center">
                <span
                  class="text-xs font-bold drop-shadow-md"
                  :class="{
                    'text-success': metric.percentage < 60,
                    'text-warning': metric.percentage >= 60 && metric.percentage < 80,
                    'text-error': metric.percentage >= 80,
                  }"
                >
                  {{ metric.percentage }}%
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Spazio per contenuto aggiuntivo con liquid glass intensificato -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 px-2">
      <div
        class="card shadow-2xl border border-white/30 bg-gradient-to-br from-white/15 to-white/8 backdrop-blur-2xl backdrop-saturate-200 hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25),0_0_20px_rgba(255,255,255,0.1)] hover:scale-[1.01] hover:border-white/40 transition-all duration-300 overflow-hidden relative before:absolute before:inset-0 before:bg-gradient-to-r before:from-transparent before:via-white/5 before:to-transparent before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300"
      >
        <div class="card-body relative z-10">
          <h2 class="card-title text-base-content drop-shadow-sm">
            <IconifyIcon icon="material-symbols:history" class="text-xl drop-shadow-sm" />
            Attività Recente
          </h2>
          <p class="text-base-content/70 text-sm">
            Le ultime operazioni sulle VM verranno mostrate qui.
          </p>
          <div class="mt-4 space-y-2">
            <div
              v-for="vm in vms.slice(0, 3)"
              :key="vm.id"
              class="flex items-center gap-3 p-2 rounded-lg bg-white/10 backdrop-blur-sm border border-white/20 min-w-0 shadow-sm"
            >
              <div
                class="w-2 h-2 rounded-full flex-shrink-0 shadow-sm"
                :class="{
                  'bg-success shadow-success/50': vm.status === 'running',
                  'bg-error shadow-error/50': vm.status === 'stopped',
                  'bg-warning shadow-warning/50': vm.status === 'unknown',
                }"
              ></div>
              <span class="text-sm truncate drop-shadow-sm">VM #{{ vm.id }} - {{ vm.status }}</span>
            </div>
          </div>
        </div>
      </div>

      <div
        class="card shadow-2xl border border-white/30 bg-gradient-to-br from-white/15 to-white/8 backdrop-blur-2xl backdrop-saturate-200 hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25),0_0_20px_rgba(255,255,255,0.1)] hover:scale-[1.01] hover:border-white/40 transition-all duration-300 overflow-hidden relative before:absolute before:inset-0 before:bg-gradient-to-r before:from-transparent before:via-white/5 before:to-transparent before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300"
      >
        <div class="card-body relative z-10">
          <h2 class="card-title text-base-content drop-shadow-sm">
            <IconifyIcon icon="material-symbols:analytics" class="text-xl drop-shadow-sm" />
            Statistiche Rapide
          </h2>
          <div class="mt-4 space-y-3">
            <div
              class="stat bg-white/10 backdrop-blur-sm border border-white/20 rounded-lg p-3 shadow-sm"
            >
              <div class="stat-title text-xs opacity-70">VM Totali</div>
              <div class="stat-value text-lg drop-shadow-sm">{{ vms.length }}</div>
            </div>
            <div
              class="stat bg-white/10 backdrop-blur-sm border border-white/20 rounded-lg p-3 shadow-sm"
            >
              <div class="stat-title text-xs opacity-70">VM Attive</div>
              <div class="stat-value text-lg text-success drop-shadow-sm">
                {{ vms.filter((vm) => vm.status === 'running').length }}
              </div>
            </div>
            <div
              class="stat bg-white/10 backdrop-blur-sm border border-white/20 rounded-lg p-3 shadow-sm"
            >
              <div class="stat-title text-xs opacity-70">Reti Create</div>
              <div class="stat-value text-lg drop-shadow-sm">{{ nets.length }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Sezione Log a tutta larghezza -->
    <div class="mt-8 px-2">
      <div
        class="card shadow-2xl border border-white/30 bg-gradient-to-br from-white/15 to-white/8 backdrop-blur-2xl backdrop-saturate-200 hover:shadow-[0_25px_50px_-12px_rgba(0,0,0,0.25),0_0_20px_rgba(255,255,255,0.1)] hover:border-white/40 transition-all duration-300 overflow-hidden relative before:absolute before:inset-0 before:bg-gradient-to-r before:from-transparent before:via-white/5 before:to-transparent before:opacity-0 hover:before:opacity-100 before:transition-opacity before:duration-300"
      >
        <div class="card-body relative z-10">
          <h2 class="card-title text-base-content drop-shadow-sm mb-4">
            <IconifyIcon icon="material-symbols:terminal" class="text-xl drop-shadow-sm" />
            Log Sistema
            <div class="badge badge-primary badge-sm ml-2">{{ logs.length }} eventi</div>
          </h2>

          <!-- Lista dei log -->
          <div class="max-h-96 overflow-y-auto pr-2 custom-scrollbar">
            <div class="space-y-2">
              <div
                v-for="log in logs"
                :key="log.id"
                class="flex items-start gap-3 p-3 rounded-lg bg-white/10 backdrop-blur-sm border border-white/20 shadow-sm hover:bg-white/15 transition-all duration-200 group"
              >
                <!-- Indicatore di tipo log -->
                <div class="flex-shrink-0 mt-1">
                  <div
                    class="w-3 h-3 rounded-full shadow-sm"
                    :class="{
                      'bg-success shadow-success/50': log.type === 'success',
                      'bg-info shadow-info/50': log.type === 'info',
                      'bg-warning shadow-warning/50': log.type === 'warning',
                      'bg-error shadow-error/50': log.type === 'error',
                    }"
                  ></div>
                </div>

                <!-- Contenuto del log -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between mb-1">
                    <span
                      class="text-xs font-mono text-base-content/60 bg-base-300/30 px-2 py-1 rounded"
                    >
                      {{ log.timestamp }}
                    </span>
                    <div class="flex items-center gap-2">
                      <span
                        class="text-xs font-semibold px-2 py-1 rounded-full border"
                        :class="{
                          'text-success border-success/30 bg-success/10': log.type === 'success',
                          'text-info border-info/30 bg-info/10': log.type === 'info',
                          'text-warning border-warning/30 bg-warning/10': log.type === 'warning',
                          'text-error border-error/30 bg-error/10': log.type === 'error',
                        }"
                      >
                        {{ log.type.toUpperCase() }}
                      </span>
                      <span class="text-xs text-base-content/50">{{ log.user }}</span>
                    </div>
                  </div>
                  <p class="text-sm text-base-content drop-shadow-sm font-medium">
                    {{ log.message }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- Footer con controlli -->
          <div class="flex items-center justify-between mt-4 pt-4 border-t border-white/20">
            <div class="flex items-center gap-2">
              <button class="btn btn-sm btn-ghost bg-white/10 border-white/20 hover:bg-white/20">
                <IconifyIcon icon="material-symbols:refresh" class="text-base" />
                Aggiorna
              </button>
            </div>
            <div class="text-xs text-base-content/60">
              Ultimo aggiornamento: {{ new Date().toLocaleTimeString('it-IT') }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar {
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.3) rgba(255, 255, 255, 0.1);
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.3);
  border-radius: 3px;
  backdrop-filter: blur(4px);
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.4);
}
</style>
