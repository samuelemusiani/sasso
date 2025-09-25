<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { Net } from '@/types'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const nets = ref<Net[]>([])
const newNetName = ref('')
const newNetVlanAware = ref(false)
const isLoading = ref(true)
const showCreateForm = ref(false)

// Statistiche computate
const netStats = computed(() => {
  const total = nets.value.length
  const ready = nets.value.filter(net => net.status === 'ready').length
  const creating = nets.value.filter(net => net.status === 'creating').length
  const vlanaware = nets.value.filter(net => net.vlanaware).length
  
  return { total, ready, creating, vlanaware }
})

async function fetchNets() {
  try {
    isLoading.value = true
    const response = await api.get('/net')
    nets.value = response.data as Net[]
  } catch (error) {
    console.error('Errore nel recuperare le reti:', error)
  } finally {
    isLoading.value = false
  }
}

// Funzione per ottenere l'icona dello stato
function getStatusIcon(status: string) {
  switch (status) {
    case 'ready': return 'material-symbols:check-circle'
    case 'creating': return 'material-symbols:pending'
    case 'error': return 'material-symbols:error'
    default: return 'material-symbols:help-circle'
  }
}

// Funzione per ottenere il colore dello stato
function getStatusColor(status: string) {
  switch (status) {
    case 'ready': return 'text-success'
    case 'creating': return 'text-warning'
    case 'error': return 'text-error'
    default: return 'text-base-content'
  }
}

async function createNet() {
  if (!newNetName.value.trim()) {
    globalNotifications.showWarning('Inserisci un nome per la rete')
    return
  }
  
  try {
    await api.post('/net', { 
      name: newNetName.value.trim(),
      vlanaware: newNetVlanAware.value
    })
    
    // Reset form
    newNetName.value = ''
    newNetVlanAware.value = false
    showCreateForm.value = false
    
    await fetchNets()
    globalNotifications.showSuccess('Rete creata con successo!')
  } catch (error) {
    console.error('Errore nella creazione della rete:', error)
    globalNotifications.showError('Errore nella creazione della rete')
  }
}

async function deleteNet(id: number) {
  if (!confirm('Sei sicuro di voler eliminare questa rete?')) {
    return
  }

  try {
    await api.delete(`/net/${id}`)
    console.log(`Rete ${id} eliminata con successo`)
    await fetchNets()
    globalNotifications.showSuccess('Rete eliminata con successo!')
  } catch (error) {
    console.error(`Errore nell'eliminazione della rete ${id}:`, error)
    globalNotifications.showError('Errore nell\'eliminazione della rete')
  }
}

onMounted(() => {
  fetchNets()
})
</script>

<template>
  <div class="space-y-8">
      
      <!-- Header -->
      <div class="text-center">
        <h1 class="text-4xl font-bold text-base-content mb-2">
          <Icon icon="material-symbols:network-node" class="inline mr-3 text-primary" />
          Gestione Reti
        </h1>
        <p class="text-base-content/70 text-lg">Configura e gestisci le tue reti virtuali</p>
      </div>

      <!-- Statistiche -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-4">
            <div class="btn btn-primary btn-square btn-lg">
              <Icon icon="material-symbols:network-node" class="text-2xl" />
            </div>
            <div>
              <h3 class="text-2xl font-bold text-base-content">{{ netStats.total }}</h3>
              <p class="text-base-content/70 font-semibold">Reti Totali</p>
            </div>
          </div>
        </div>

        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-4">
            <div class="btn btn-success btn-square btn-lg">
              <Icon icon="material-symbols:check-circle" class="text-2xl" />
            </div>
            <div>
              <h3 class="text-2xl font-bold text-base-content">{{ netStats.ready }}</h3>
              <p class="text-base-content/70 font-semibold">Pronte</p>
            </div>
          </div>
        </div>

        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-4">
            <div class="btn btn-warning btn-square btn-lg">
              <Icon icon="material-symbols:pending" class="text-2xl" />
            </div>
            <div>
              <h3 class="text-2xl font-bold text-base-content">{{ netStats.creating }}</h3>
              <p class="text-base-content/70 font-semibold">In Creazione</p>
            </div>
          </div>
        </div>

        <div class="liquid-glass-card p-6">
          <div class="flex items-center gap-4">
            <div class="btn btn-info btn-square btn-lg">
              <Icon icon="material-symbols:lan" class="text-2xl" />
            </div>
            <div>
              <h3 class="text-2xl font-bold text-base-content">{{ netStats.vlanaware }}</h3>
              <p class="text-base-content/70 font-semibold">VLAN Aware</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Form Creazione Rete -->
      <div class="liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-info/5">
        <div class="p-6">
          <button @click="showCreateForm = !showCreateForm"
                  class="w-full flex items-center justify-between text-left">
            <div class="flex items-center gap-3">
              <Icon icon="material-symbols:add-circle" class="text-2xl text-primary" />
              <h2 class="text-xl font-bold text-base-content">Crea Nuova Rete</h2>
            </div>
            <Icon :icon="showCreateForm ? 'material-symbols:keyboard-arrow-up' : 'material-symbols:keyboard-arrow-down'" 
                  class="text-2xl text-base-content/50 transition-transform duration-200" />
          </button>
        </div>

        <div v-if="showCreateForm" class="border-t border-white/20">
          <div class="p-6 space-y-6">
            <!-- Campi del form -->
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <!-- Nome Rete -->
              <div class="form-control mb-4">
                <span class="label-text text-base-content font-semibold block mb-3">Nome Rete</span>
                <input v-model="newNetName" 
                       type="text"
                       placeholder="es. network-produzione"
                       class="input input-bordered bg-base-100/10 border-white/30 focus:border-primary/50 text-base-content placeholder:text-base-content/50 w-full" />
                <span class="label-text-alt text-base-content/50 block mt-2">
                  Subnet e gateway verranno assegnati automaticamente
                </span>
              </div>

              <!-- VLAN Aware -->
              <div class="form-control mb-4">
                <span class="label-text text-base-content font-semibold block mb-3">VLAN Aware</span>
                <label class="cursor-pointer flex items-center gap-3">
                  <input v-model="newNetVlanAware" 
                         type="checkbox" 
                         class="checkbox checkbox-primary" />
                  <span class="label-text text-base-content">Abilita supporto VLAN</span>
                </label>
                <span class="label-text-alt text-base-content/50 block mt-2">
                  Abilita supporto VLAN per la rete
                </span>
              </div>
            </div>

            <!-- Pulsanti -->
            <div class="flex justify-end gap-3 pt-2">
              <button @click="showCreateForm = false" 
                      class="btn btn-ghost">
                <Icon icon="material-symbols:close" class="text-lg" />
                Annulla
              </button>
              <button @click="createNet" 
                      class="btn btn-primary">
                <Icon icon="material-symbols:add" class="text-lg" />
                Crea Rete
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Lista Reti -->
      <div>
        <h2 class="text-xl font-bold text-base-content mb-4">Le tue Reti</h2>
        
        <div v-if="nets.length === 0" class="liquid-glass-card-no-scale p-8 text-center">
          <Icon icon="material-symbols:wifi-off" class="text-6xl text-base-content/30 mb-4" />
          <h3 class="text-lg font-semibold text-base-content/70 mb-2">Nessuna rete trovata</h3>
          <p class="text-base-content/50">Crea la tua prima rete per iniziare!</p>
        </div>

        <!-- Network Cards -->
        <div v-else class="space-y-4">
          <div v-for="net in nets" :key="net.id" class="liquid-glass-card-no-scale p-6">
            <div class="flex items-center justify-between">
              <!-- Informazioni Rete -->
              <div class="flex items-center gap-6">
                <!-- Stato -->
                <div class="flex items-center gap-3">
                  <div class="btn btn-square btn-md rounded-xl p-0 m-1 flex-shrink-0"
                       :class="{
                         'btn-success': net.status === 'ready',
                         'btn-warning': net.status === 'creating',
                         'btn-error': net.status === 'error',
                         'btn-neutral': !['ready', 'creating', 'error'].includes(net.status)
                       }">
                    <Icon :icon="getStatusIcon(net.status)" class="text-xl" />
                  </div>
                  <div>
                    <h3 class="font-bold text-lg text-base-content">
                      {{ net.name || `Rete #${net.id}` }}
                    </h3>
                    <span class="text-sm font-semibold capitalize" :class="getStatusColor(net.status)">
                      {{ net.status }}
                    </span>
                  </div>
                </div>

                <!-- Dettagli -->
                <div class="hidden md:flex items-center gap-6">
                  <div class="text-center">
                    <p class="text-sm text-base-content/50 font-medium">ID</p>
                    <p class="text-base font-bold text-base-content">{{ net.id }}</p>
                  </div>
                  <div class="text-center">
                    <p class="text-sm text-base-content/50 font-medium">Subnet</p>
                    <p class="text-sm font-mono text-base-content">{{ net.subnet || 'N/A' }}</p>
                  </div>
                  <div class="text-center">
                    <p class="text-sm text-base-content/50 font-medium">Gateway</p>
                    <p class="text-sm font-mono text-base-content">{{ net.gateway || 'N/A' }}</p>
                  </div>
                  <div class="text-center">
                    <p class="text-sm text-base-content/50 font-medium">VLAN Aware</p>
                    <div class="flex items-center justify-center">
                      <Icon :icon="net.vlanaware ? 'material-symbols:check-circle' : 'material-symbols:cancel'" 
                            :class="net.vlanaware ? 'text-success' : 'text-base-content/30'" 
                            class="text-lg" />
                    </div>
                  </div>
                </div>
              </div>
              
              <!-- Azioni -->
              <div class="flex items-center gap-2">
                <button v-if="net.status === 'ready'" @click="deleteNet(net.id)"
                        class="btn btn-error btn-sm">
                  <Icon icon="material-symbols:delete" class="text-lg" />
                  <span class="hidden sm:inline ml-1">Elimina</span>
                </button>
              </div>
            </div>

            <!-- Dettagli Mobile -->
            <div class="md:hidden mt-4 pt-4 border-t border-white/20">
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="text-base-content/50">ID: </span>
                  <span class="font-semibold text-base-content">{{ net.id }}</span>
                </div>
                <div>
                  <span class="text-base-content/50">VLAN Aware: </span>
                  <Icon :icon="net.vlanaware ? 'material-symbols:check-circle' : 'material-symbols:cancel'" 
                        :class="net.vlanaware ? 'text-success' : 'text-base-content/30'" 
                        class="text-base" />
                </div>
                <div>
                  <span class="text-base-content/50">Subnet: </span>
                  <span class="font-mono text-xs text-base-content">{{ net.subnet || 'N/A' }}</span>
                </div>
                <div>
                  <span class="text-base-content/50">Gateway: </span>
                  <span class="font-mono text-xs text-base-content">{{ net.gateway || 'N/A' }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
  </div>
</template>
