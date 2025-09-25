<script setup lang="ts">
import { api } from '@/lib/api'
import { onMounted, ref, computed } from 'vue'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const vpnConfig = ref('')
const isLoading = ref(true)
const copySuccess = ref(false)

// Statistiche computate
const vpnStats = computed(() => {
  const lines = vpnConfig.value.split('\n').filter(line => line.trim()).length
  const hasConfig = vpnConfig.value.length > 0
  
  return { lines, hasConfig }
})

// Funzione per mascherare le chiavi private nella configurazione
const maskedConfig = computed(() => {
  if (!vpnConfig.value) return ''
  
  return vpnConfig.value.replace(
    /(PrivateKey\s*=\s*)([A-Za-z0-9+/=]+)/g, 
    '$1' + '•'.repeat(35)
  ).replace(
    /(PresharedKey\s*=\s*)([A-Za-z0-9+/=]+)/g, 
    '$1' + '•'.repeat(35)
  )
})

async function fetchVPNConfig() {
  try {
    isLoading.value = true
    const response = await api.get('/vpn')
    vpnConfig.value = response.data as string
    console.log('VPN Config response:', response.data)
  } catch (error) {
    console.error('Errore nel recuperare la configurazione VPN:', error)
  } finally {
    isLoading.value = false
  }
}

// Funzione per copiare la configurazione negli appunti
async function copyConfig() {
  try {
    await navigator.clipboard.writeText(vpnConfig.value)
    copySuccess.value = true
    setTimeout(() => {
      copySuccess.value = false
    }, 2000)
  } catch (error) {
    console.error('Errore nella copia:', error)
    globalNotifications.showError('Errore nella copia della configurazione')
  }
}

// Funzione per scaricare il file di configurazione
function downloadConfig() {
  if (!vpnConfig.value) {
    globalNotifications.showWarning('Nessuna configurazione disponibile per il download')
    return
  }
  
  const blob = new Blob([vpnConfig.value], { type: 'text/plain' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'sasso-wireguard.conf'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

onMounted(() => {
  fetchVPNConfig()
})
</script>

<template>
  <div class="space-y-8">
    
    <!-- Header -->
    <div class="text-center">
      <h1 class="text-4xl font-bold text-base-content mb-2">
        <Icon icon="material-symbols:vpn-key" class="inline mr-3 text-primary" />
        Configurazione VPN
      </h1>
      <p class="text-base-content/70 text-lg">Scarica e configura il tuo accesso VPN WireGuard</p>
    </div>



    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg text-primary"></span>
    </div>

    <!-- Configurazione VPN -->
    <div v-else-if="vpnConfig" class="liquid-glass-card">
      <div class="card-body p-6">
        <div class="flex items-center justify-between mb-6">
          <h2 class="card-title text-base-content flex items-center gap-3">
            <Icon icon="material-symbols:settings" class="text-primary text-2xl" />
            File di Configurazione WireGuard
          </h2>
          
          <!-- Pulsanti di azione -->
          <div class="flex items-center gap-2">
            <button 
              @click="copyConfig()"
              class="btn btn-outline btn-sm gap-2"
              :class="copySuccess ? 'btn-success' : 'btn-primary'"
            >
              <Icon :icon="copySuccess ? 'material-symbols:check' : 'material-symbols:content-copy'" class="text-lg" />
              {{ copySuccess ? 'Copiato!' : 'Copia' }}
            </button>
            
            <button 
              @click="downloadConfig()"
              class="btn btn-primary btn-sm gap-2"
            >
              <Icon icon="material-symbols:download" class="text-lg" />
              Download .conf
            </button>
          </div>
        </div>
        
        <!-- Area configurazione -->
        <div class="bg-base-200/50 rounded-lg p-4 border border-base-300/50">
          <div class="flex items-center justify-between mb-2">
            <p class="text-xs text-base-content/60 font-semibold">sasso-wireguard.conf</p>
            <div class="flex items-center gap-2">
              <div class="badge badge-warning badge-sm">
                <Icon icon="material-symbols:visibility-off" class="text-xs mr-1" />
                Chiavi Mascherate
              </div>
              <div class="badge badge-outline badge-sm">
                <Icon icon="material-symbols:code" class="text-xs mr-1" />
                {{ vpnStats.lines }} linee
              </div>
            </div>
          </div>
          <pre class="text-sm font-mono text-base-content whitespace-pre-wrap break-all leading-relaxed max-h-96 overflow-y-auto">{{ maskedConfig }}</pre>
        </div>
        
        <!-- Istruzioni -->
        <div class="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="bg-info/10 rounded-lg p-4 border border-info/20">
            <h3 class="font-semibold text-info mb-2 flex items-center gap-2">
              <Icon icon="material-symbols:info" class="text-lg" />
              Come utilizzare
            </h3>
            <ul class="text-sm text-base-content/80 space-y-1">
              <li>• Scarica il file .conf</li>
              <li>• Importa in WireGuard</li>
              <li>• Attiva la connessione</li>
            </ul>
          </div>
          
          <div class="bg-success/10 rounded-lg p-4 border border-success/20">
            <h3 class="font-semibold text-success mb-2 flex items-center gap-2">
              <Icon icon="material-symbols:content-copy" class="text-lg" />
              Copia e Download
            </h3>
            <ul class="text-sm text-base-content/80 space-y-1">
              <li>• Le chiavi sono mascherate qui</li>
              <li>• Copia/Download in chiaro</li>
              <li>• Configurazione completa</li>
            </ul>
          </div>
          
          <div class="bg-warning/10 rounded-lg p-4 border border-warning/20">
            <h3 class="font-semibold text-warning mb-2 flex items-center gap-2">
              <Icon icon="material-symbols:security" class="text-lg" />
              Sicurezza
            </h3>
            <ul class="text-sm text-base-content/80 space-y-1">
              <li>• Non condividere il file</li>
              <li>• Mantieni privata la chiave</li>
              <li>• Usa solo su dispositivi fidati</li>
            </ul>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="text-center py-12">
      <div class="liquid-glass-card p-8">
        <Icon icon="material-symbols:vpn-key-off" class="text-6xl text-base-content/30 mx-auto mb-4" />
        <h3 class="text-xl font-semibold text-base-content mb-2">Configurazione VPN Non Disponibile</h3>
        <p class="text-base-content/70">La configurazione VPN non è ancora stata generata o non è disponibile.</p>
        <button 
          @click="fetchVPNConfig()"
          class="btn btn-primary mt-4 gap-2"
        >
          <Icon icon="material-symbols:refresh" class="text-lg" />
          Riprova
        </button>
      </div>
    </div>
  </div>
</template>
