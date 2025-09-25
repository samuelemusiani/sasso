<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { VM, Interface, Net } from '@/types'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const vms = ref<VM[]>([])
const vmName = ref('')
const vmTemplate = ref('ubuntu-22.04')
const cores = ref(1)
const ram = ref(1024)
const disk = ref(4)
const include_global_ssh_keys = ref(true)
const isLoading = ref(true)
const showCreateForm = ref(false)
const expandedSettings = ref<Record<number, boolean>>({})
const expandedInterfaces = ref<Record<number, boolean>>({})

// Interfacce e reti
const vmInterfaces = ref<Record<number, Interface[]>>({})
const nets = ref<Net[]>([])

// Mappa delle reti per nome
const netMap = computed(() => {
  const map = new Map<number, string>()
  for (const net of nets.value) {
    map.set(net.id, net.name)
  }
  return map
})

// Campi per le impostazioni VM
const editingVM = ref<Record<number, {
  name: string
  startAtBoot: boolean
  includeGlobalSSHKeys: boolean
}>>({})

// Template disponibili
const availableTemplates = [
  { value: 'ubuntu-22.04', label: 'Ubuntu 22.04 LTS' },
  { value: 'ubuntu-20.04', label: 'Ubuntu 20.04 LTS' }, 
  { value: 'debian-12', label: 'Debian 12' },
  { value: 'centos-9', label: 'CentOS Stream 9' },
  { value: 'alpine-3.18', label: 'Alpine Linux 3.18' },
  { value: 'rocky-9', label: 'Rocky Linux 9' }
]

// Statistiche computate
const vmStats = computed(() => {
  const total = vms.value.length
  const running = vms.value.filter(vm => vm.status === 'running').length
  const stopped = vms.value.filter(vm => vm.status === 'stopped').length
  const other = total - running - stopped
  
  return { total, running, stopped, other }
})

async function fetchVMs() {
  try {
    const response = await api.get('/vm')
    vms.value = response.data as VM[]
  } catch (error) {
    console.error('Errore nel recuperare le VM:', error)
  }
}

// Funzione per ottenere l'icona dello stato
function getStatusIcon(status: string) {
  switch (status) {
    case 'running': return 'material-symbols:play-circle'
    case 'stopped': return 'material-symbols:stop-circle'
    case 'starting': return 'material-symbols:pending'
    default: return 'material-symbols:help-circle'
  }
}

// Funzione per ottenere il colore dello stato
function getStatusColor(status: string) {
  switch (status) {
    case 'running': return 'text-success'
    case 'stopped': return 'text-error'
    case 'starting': return 'text-warning'
    default: return 'text-base-content'
  }
}

async function createVM() {
  if (!vmName.value.trim()) {
    globalNotifications.showError('Campo obbligatorio', 'Inserisci un nome per la VM')
    return
  }
  
  try {
    await api.post('/vm', {
      name: vmName.value.trim(),
      template: vmTemplate.value,
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
      include_global_ssh_keys: include_global_ssh_keys.value,
    })
    
    // Reset form
    vmName.value = ''
    vmTemplate.value = 'ubuntu-22.04'
    cores.value = 1
    ram.value = 1024
    disk.value = 4
    include_global_ssh_keys.value = true
    showCreateForm.value = false
    
    await fetchVMs()
    globalNotifications.showSuccess('VM creata', 'La macchina virtuale è stata creata con successo!')
  } catch (error) {
    console.error('Errore nella creazione della VM:', error)
    globalNotifications.showError('Errore creazione VM', 'Impossibile creare la macchina virtuale')
  }
}

async function deleteVM(vmid: number) {
  console.log('deleteVM called for VM:', vmid)
  if (confirm(`Sei sicuro di voler eliminare la VM #${vmid}?`)) {
    try {
      await api.delete(`/vm/${vmid}`)
      await fetchVMs()
      globalNotifications.showSuccess('VM eliminata', `La macchina virtuale #${vmid} è stata eliminata`)
      console.log(`VM #${vmid} eliminata con successo`)
    } catch (error) {
      console.error('Errore nell\'eliminazione della VM:', error)
      globalNotifications.showError('Errore eliminazione', 'Impossibile eliminare la macchina virtuale')
    }
  }
}

async function startVM(vmid: number) {
  console.log('startVM called for VM:', vmid)
  try {
    await api.post(`/vm/${vmid}/start`)
    await fetchVMs()
    globalNotifications.showSuccess('VM avviata', `La macchina virtuale #${vmid} è stata avviata`)
    console.log(`VM #${vmid} avviata con successo`)
  } catch (error) {
    console.error('Errore nell\'avvio della VM:', error)
    globalNotifications.showError('Errore avvio', 'Impossibile avviare la macchina virtuale')
  }
}

async function stopVM(vmid: number) {
  try {
    await api.post(`/vm/${vmid}/stop`)
    await fetchVMs()
    globalNotifications.showSuccess('VM fermata', `La macchina virtuale #${vmid} è stata fermata`)
    console.log(`VM #${vmid} fermata con successo`)
  } catch (error) {
    console.error('Errore nel fermare la VM:', error)
    globalNotifications.showError('Errore stop', 'Impossibile fermare la macchina virtuale')
  }
}

async function restartVM(vmid: number) {
  try {
    await api.post(`/vm/${vmid}/restart`)
    await fetchVMs()
    globalNotifications.showSuccess('VM riavviata', `La macchina virtuale #${vmid} è stata riavviata`)
    console.log(`VM #${vmid} riavviata con successo`)
  } catch (error) {
    console.error('Errore nel riavvio della VM:', error)
    globalNotifications.showError('Errore riavvio', 'Impossibile riavviare la macchina virtuale')
  }
}

// Funzioni per gestire le sezioni espandibili
function toggleSettings(vmid: number) {
  expandedSettings.value[vmid] = !expandedSettings.value[vmid]
  
  // Inizializza i dati di editing se non esistono
  if (expandedSettings.value[vmid] && !editingVM.value[vmid]) {
    const vm = vms.value.find(v => v.id === vmid)
    if (vm) {
      editingVM.value[vmid] = {
        name: vm.name || '',
        startAtBoot: false, // TODO: aggiungere questo campo al tipo VM
        includeGlobalSSHKeys: vm.include_global_ssh_keys
      }
    }
  }
}

async function toggleInterfaces(vmid: number) {
  expandedInterfaces.value[vmid] = !expandedInterfaces.value[vmid]
  
  // Carica le interfacce quando viene espansa la sezione
  if (expandedInterfaces.value[vmid] && !vmInterfaces.value[vmid]) {
    await fetchVMInterfaces(vmid)
  }
}

async function fetchVMInterfaces(vmid: number) {
  try {
    const response = await api.get(`/vm/${vmid}/interface`)
    vmInterfaces.value[vmid] = response.data as Interface[]
  } catch (error) {
    console.error('Errore nel recuperare le interfacce della VM:', error)
    vmInterfaces.value[vmid] = []
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

async function updateVMSettings(vmid: number) {
  if (!editingVM.value[vmid]) return
  
  try {
    await api.put(`/vm/${vmid}`, {
      name: editingVM.value[vmid].name,
      include_global_ssh_keys: editingVM.value[vmid].includeGlobalSSHKeys,
      // start_at_boot: editingVM.value[vmid].startAtBoot // TODO: implementare nel backend
    })
    
    await fetchVMs()
    expandedSettings.value[vmid] = false
    globalNotifications.showSuccess('Impostazioni aggiornate con successo!')
  } catch (error) {
    console.error('Errore nell\'aggiornamento delle impostazioni:', error)
    globalNotifications.showError('Errore nell\'aggiornamento delle impostazioni')
  }
}

onMounted(async () => {
  await Promise.all([fetchVMs(), fetchNets()])
  isLoading.value = false

  // Aggiornamento periodico ogni 10 secondi
  const interval = setInterval(() => {
    fetchVMs()
  }, 10000)

  // Cleanup quando il componente viene distrutto
  return () => clearInterval(interval)
})
</script>

<template>
  <div class="space-y-8">
      
      <!-- Header -->
      <div class="text-center">
        <h1 class="text-4xl font-bold text-base-content mb-2">
          <Icon icon="material-symbols:computer" class="inline mr-3 text-primary" />
          Gestione Macchine Virtuali
        </h1>
        <p class="text-base-content/70 text-lg">Crea, gestisci e monitora le tue VM</p>
      </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center items-center h-64">
      <div class="loading loading-spinner loading-lg"></div>
      <span class="ml-4 text-lg">Caricamento VM...</span>
    </div>

      <!-- Statistiche VM -->
      <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-6">
        <!-- Totale VM -->
        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-3">
            <div class="btn btn-square btn-sm rounded-xl btn-primary p-0 flex-shrink-0">
              <Icon icon="material-symbols:computer" class="text-lg" />
            </div>
            <div>
              <h3 class="font-bold text-base text-base-content">Totale</h3>
              <span class="text-sm font-semibold text-base-content/80">
                {{ vmStats.total }} VM
              </span>
            </div>
          </div>
        </div>

        <!-- VM Running -->
        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-4">
            <div class="btn btn-success btn-square btn-lg">
              <Icon icon="material-symbols:play-circle" class="text-2xl" />
            </div>
            <div>
              <h3 class="text-2xl font-bold text-base-content">{{ vmStats.running }}</h3>
              <p class="text-base-content/70 font-semibold">In Esecuzione</p>
            </div>
          </div>
        </div>

        <!-- VM Stopped -->
        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-4">
            <div class="btn btn-error btn-square btn-lg">
              <Icon icon="material-symbols:stop-circle" class="text-2xl" />
            </div>
            <div>
              <h3 class="text-2xl font-bold text-base-content">{{ vmStats.stopped }}</h3>
              <p class="text-base-content/70 font-semibold">Ferme</p>
            </div>
          </div>
        </div>

        <!-- Crea Nuova VM -->
        <div class="liquid-glass-card p-6 cursor-pointer hover:scale-105 transition-transform" @click="showCreateForm = !showCreateForm">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <div class="btn btn-info btn-square btn-lg">
                <Icon icon="material-symbols:add-circle" class="text-2xl" />
              </div>
              <div>
                <h3 class="text-2xl font-bold text-base-content">Nuova VM</h3>
                <p class="text-base-content/70 font-semibold">Clicca per creare</p>
              </div>
            </div>
            <Icon :icon="showCreateForm ? 'material-symbols:expand-less' : 'material-symbols:expand-more'" 
                  class="text-2xl text-base-content/50" />
          </div>
        </div>
      </div>

      <!-- Form Creazione VM (collassabile) -->
      <div v-if="showCreateForm" class="liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-info/5 p-6">
        <h2 class="text-xl font-bold text-base-content mb-6 flex items-center gap-3">
          <Icon icon="material-symbols:add-circle" class="text-2xl text-primary" />
          Crea Nuova Macchina Virtuale
        </h2>
        
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-6">
          <!-- Nome VM -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-semibold">Nome VM</span>
            </label>
            <input type="text" v-model="vmName" placeholder="es. web-server-01" 
                   class="input input-bordered w-full" required />
          </div>
          
          <!-- Template -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-semibold">Sistema Operativo</span>
            </label>
            <select v-model="vmTemplate" class="select select-bordered w-full">
              <option v-for="template in availableTemplates" :key="template.value" :value="template.value">
                {{ template.label }}
              </option>
            </select>
          </div>
          
          <!-- Cores -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-semibold">CPU Cores</span>
            </label>
            <input type="number" v-model="cores" min="1" max="16" 
                   class="input input-bordered w-full" />
          </div>
          
          <!-- RAM -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-semibold">RAM (MB)</span>
            </label>
            <input type="number" v-model="ram" min="512" max="32768" step="512"
                   class="input input-bordered w-full" />
          </div>
          
          <!-- Disk -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-semibold">Disco (GB)</span>
            </label>
            <input type="number" v-model="disk" min="1" max="1000" 
                   class="input input-bordered w-full" />
          </div>
          
          <!-- SSH Keys -->
          <div class="form-control">
            <label class="label">
              <span class="label-text font-semibold">SSH Keys</span>
            </label>
            <div class="flex items-center h-12">
              <input type="checkbox" v-model="include_global_ssh_keys" 
                     class="checkbox checkbox-primary mr-3" />
              <span class="text-sm">Includi chiavi globali</span>
            </div>
          </div>
        </div>

        <!-- Alert informativi -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
          <div class="alert alert-info">
            <Icon icon="material-symbols:info" class="text-xl" />
            <div>
              <h4 class="font-bold">Informazione</h4>
              <p class="text-sm">Le chiavi SSH globali permettono un troubleshooting migliore in caso di problemi.</p>
            </div>
          </div>
          
          <div class="alert alert-warning">
            <Icon icon="material-symbols:warning" class="text-xl" />
            <div>
              <h4 class="font-bold">Attenzione</h4>
              <p class="text-sm">Stop e restart sono operazioni immediate, non spegnimenti graziosi.</p>
            </div>
          </div>
        </div>
        
        <!-- Pulsanti azione -->
        <div class="flex gap-4 justify-end">
          <button class="btn btn-ghost" @click="showCreateForm = false">
            <Icon icon="material-symbols:cancel" class="text-lg mr-2" />
            Annulla
          </button>
          <button class="btn btn-primary" @click="createVM()">
            <Icon icon="material-symbols:add" class="text-lg mr-2" />
            Crea VM
          </button>
        </div>
      </div>

      <!-- Lista VM -->
      <div>
        <h2 class="text-xl font-bold text-base-content mb-4">Le tue Macchine Virtuali</h2>
        
        <div v-if="vms.length === 0" class="liquid-glass-card-no-scale p-8 text-center">
          <Icon icon="material-symbols:computer-off" class="text-6xl text-base-content/30 mb-4" />
          <h3 class="text-lg font-semibold text-base-content/70 mb-2">Nessuna VM trovata</h3>
          <p class="text-base-content/50">Crea la tua prima macchina virtuale per iniziare!</p>
        </div>

        <!-- VM Cards -->
        <div v-else class="space-y-4">
          <div v-for="vm in vms" :key="vm.id" class="liquid-glass-card-no-scale p-6">
            <div class="flex items-center justify-between">
              <!-- Informazioni VM -->
              <div class="flex items-center gap-6">
                <!-- Stato -->
                <div class="flex items-center gap-3">
                  <div class="btn btn-square btn-md rounded-xl p-0 m-1 flex-shrink-0"
                       :class="{
                         'btn-success': vm.status === 'running',
                         'btn-error': vm.status === 'stopped',
                         'btn-warning': vm.status === 'starting',
                         'btn-neutral': !['running', 'stopped', 'starting'].includes(vm.status)
                       }">
                    <Icon :icon="getStatusIcon(vm.status)" class="text-xl" />
                  </div>
                  <div>
                    <h3 class="font-bold text-lg text-base-content">
                      {{ vm.name || `VM #${vm.id}` }}
                    </h3>
                    <span class="text-sm font-semibold capitalize" :class="getStatusColor(vm.status)">
                      {{ vm.status }}
                    </span>
                  </div>
                </div>
                
                <!-- Specifiche -->
                <div class="hidden lg:flex items-center gap-6 text-sm text-base-content/70">
                  <div class="flex items-center gap-2">
                    <Icon icon="material-symbols:memory" class="text-lg" />
                    <span>{{ vm.cores }} cores</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <Icon icon="material-symbols:storage" class="text-lg" />
                    <span>{{ vm.ram }} MB</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <Icon icon="material-symbols:hard-drive" class="text-lg" />
                    <span>{{ vm.disk }} GB</span>
                  </div>
                </div>
              </div>
              
              <!-- Azioni -->
              <div class="flex items-center gap-2">
                <button @click="toggleSettings(vm.id)"
                        class="btn btn-info btn-sm">
                  <Icon icon="material-symbols:settings" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Impostazioni</span>
                </button>
                
                <button @click="toggleInterfaces(vm.id)"
                        class="btn btn-primary btn-sm">
                  <Icon icon="material-symbols:network-node" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Interfacce</span>
                </button>
                
                <button v-if="vm.status === 'stopped'" @click="startVM(vm.id)"
                        class="btn btn-success btn-sm">
                  <Icon icon="material-symbols:play-arrow" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Avvia</span>
                </button>
                
                <button v-if="vm.status === 'running'" @click="stopVM(vm.id)"
                        class="btn btn-warning btn-sm">
                  <Icon icon="material-symbols:stop" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Ferma</span>
                </button>
                
                <button v-if="vm.status === 'running'" @click="restartVM(vm.id)"
                        class="btn btn-info btn-sm">
                  <Icon icon="material-symbols:restart-alt" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Riavvia</span>
                </button>
                
                <button @click="deleteVM(vm.id)" class="btn btn-error btn-sm">
                  <Icon icon="material-symbols:delete" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Elimina</span>
                </button>
              </div>
            </div>
            
            <!-- Specifiche mobile -->
            <div class="lg:hidden mt-4 pt-4 border-t border-white/10">
              <div class="flex items-center gap-4 text-sm text-base-content/70">
                <div class="flex items-center gap-2">
                  <Icon icon="material-symbols:memory" class="text-lg" />
                  <span>{{ vm.cores }} cores</span>
                </div>
                <div class="flex items-center gap-2">
                  <Icon icon="material-symbols:storage" class="text-lg" />
                  <span>{{ vm.ram }} MB</span>
                </div>
                <div class="flex items-center gap-2">
                  <Icon icon="material-symbols:hard-drive" class="text-lg" />
                  <span>{{ vm.disk }} GB</span>
                </div>
              </div>
            </div>

            <!-- Sezione Impostazioni Espandibile -->
            <div v-if="expandedSettings[vm.id]" class="mt-4 pt-4 border-t border-white/20">
              <div class="liquid-glass-card-no-scale bg-gradient-to-br from-info/5 via-transparent to-primary/5 p-4">
                <div class="flex items-center gap-3 mb-4">
                  <Icon icon="material-symbols:settings" class="text-xl text-info" />
                  <h4 class="text-lg font-semibold text-base-content">Impostazioni VM</h4>
                </div>
                
                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <!-- Nome VM -->
                  <div class="form-control">
                    <label class="label">
                      <span class="label-text text-base-content font-medium">Nome VM</span>
                    </label>
                    <input v-model="editingVM[vm.id].name" 
                           type="text"
                           placeholder="Nome della VM"
                           class="input input-bordered bg-base-100/10 border-white/30 focus:border-info/50 text-base-content" />
                  </div>

                  <!-- Include Global SSH Keys -->
                  <div class="form-control">
                    <label class="label">
                      <span class="label-text text-base-content font-medium">Chiavi SSH Globali</span>
                    </label>
                    <label class="cursor-pointer flex items-center gap-3 mt-2">
                      <input v-model="editingVM[vm.id].includeGlobalSSHKeys" 
                             type="checkbox" 
                             class="checkbox checkbox-info" />
                      <span class="label-text text-base-content">Includi chiavi SSH globali</span>
                    </label>
                  </div>

                  <!-- Start at Boot (placeholder per futura implementazione) -->
                  <div class="form-control">
                    <label class="label">
                      <span class="label-text text-base-content font-medium">Avvio Automatico</span>
                    </label>
                    <label class="cursor-pointer flex items-center gap-3 mt-2">
                      <input v-model="editingVM[vm.id].startAtBoot" 
                             type="checkbox" 
                             class="checkbox checkbox-info" 
                             disabled />
                      <span class="label-text text-base-content/50">Avvia all'avvio del sistema (presto disponibile)</span>
                    </label>
                  </div>
                </div>

                <!-- Pulsanti -->
                <div class="flex justify-end gap-3 mt-6">
                  <button @click="expandedSettings[vm.id] = false" 
                          class="btn btn-ghost">
                    <Icon icon="material-symbols:close" class="text-lg" />
                    Annulla
                  </button>
                  <button @click="updateVMSettings(vm.id)" 
                          class="btn btn-info">
                    <Icon icon="material-symbols:save" class="text-lg" />
                    Salva Impostazioni
                  </button>
                </div>
              </div>
            </div>

            <!-- Sezione Interfacce Espandibile -->
            <div v-if="expandedInterfaces[vm.id]" class="mt-4 pt-4 border-t border-white/20">
              <div class="liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-accent/5 p-4">
                <div class="flex items-center justify-between mb-4">
                  <div class="flex items-center gap-3">
                    <Icon icon="material-symbols:network-node" class="text-xl text-primary" />
                    <h4 class="text-lg font-semibold text-base-content">
                      Interfacce di Rete
                      <span v-if="vmInterfaces[vm.id]" class="text-sm font-normal text-base-content/50 ml-2">
                        ({{ vmInterfaces[vm.id].length }})
                      </span>
                    </h4>
                  </div>
                  <RouterLink :to="`/vm/${vm.id}/interfaces`" 
                             class="btn btn-primary btn-sm">
                    <Icon icon="material-symbols:open-in-new" class="text-lg" />
                    Gestisci Interfacce
                  </RouterLink>
                </div>
                
                <!-- Lista Interfacce -->
                <div v-if="vmInterfaces[vm.id] && vmInterfaces[vm.id].length > 0" 
                     class="max-h-48 overflow-y-auto space-y-2 mb-4">
                  <div v-for="iface in vmInterfaces[vm.id]" :key="iface.id" 
                       class="flex items-center justify-between p-3 bg-base-100/10 rounded-lg border border-white/20">
                    <div class="flex items-center gap-3">
                      <div class="w-3 h-3 rounded-full"
                           :class="{
                             'bg-success': iface.status === 'active',
                             'bg-error': iface.status === 'inactive', 
                             'bg-warning': iface.status === 'pending',
                             'bg-base-content/30': !['active', 'inactive', 'pending'].includes(iface.status)
                           }"></div>
                      <div>
                        <p class="font-semibold text-base-content text-sm">
                          {{ netMap.get(iface.vnet_id) || `Rete ID ${iface.vnet_id}` }}
                        </p>
                        <p class="text-xs text-base-content/60">
                          IP: {{ iface.ip_add || 'N/A' }} 
                          <span v-if="iface.vlan_tag > 0" class="ml-2">VLAN: {{ iface.vlan_tag }}</span>
                        </p>
                      </div>
                    </div>
                    <div class="text-xs font-mono text-base-content/50">
                      #{{ iface.id }}
                    </div>
                  </div>
                </div>
                
                <!-- Stato vuoto -->
                <div v-else-if="vmInterfaces[vm.id] && vmInterfaces[vm.id].length === 0" 
                     class="text-center py-6">
                  <Icon icon="material-symbols:network-off" class="text-3xl text-base-content/30 mb-2" />
                  <p class="text-base-content/70 text-sm">Nessuna interfaccia configurata</p>
                  <p class="text-base-content/50 text-xs">Usa "Gestisci Interfacce" per aggiungerne una</p>
                </div>
                
                <!-- Loading -->
                <div v-else class="text-center py-6">
                  <Icon icon="material-symbols:refresh" class="text-2xl text-base-content/50 animate-spin mb-2" />
                  <p class="text-base-content/50 text-sm">Caricamento interfacce...</p>
                </div>

                <div class="flex justify-end mt-4">
                  <button @click="expandedInterfaces[vm.id] = false" 
                          class="btn btn-ghost">
                    <Icon icon="material-symbols:close" class="text-lg" />
                    Chiudi
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
  </div>
</template>
