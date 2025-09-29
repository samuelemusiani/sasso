<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import type { Interface, Net, VM } from '@/types'
import { api } from '@/lib/api'
import InterfaceForm from '@/components/vm/InterfaceForm.vue'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const route = useRoute()
const vmid = Number(route.params.vmid)

const interfaces = ref<Interface[]>([])
const nets = ref<Net[]>([])
const vm = ref<VM | null>(null)
const showAddForm = ref(false)
const editingInterface = ref<Interface | null>(null)
const isLoading = ref(true)

const netMap = computed(() => {
  const map = new Map<number, string>()
  for (const net of nets.value) {
    map.set(net.id, net.name)
  }
  return map
})

async function fetchVM() {
  try {
    const response = await api.get('/vm')
    const vms = response.data as VM[]
    vm.value = vms.find((v) => v.id === vmid) || null
  } catch (error) {
    console.error('Errore nel recuperare la VM:', error)
  }
}

async function fetchInterfaces() {
  try {
    const response = await api.get(`/vm/${vmid}/interface`)
    interfaces.value = response.data as Interface[]
  } catch (error) {
    console.error('Errore nel recuperare le interfacce:', error)
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

async function deleteInterface(ifaceid: number) {
  if (!confirm('Sei sicuro di voler eliminare questa interfaccia?')) {
    return
  }

  try {
    await api.delete(`/vm/${vmid}/interface/${ifaceid}`)
    await fetchInterfaces()
    globalNotifications.showSuccess('Interfaccia eliminata con successo!')
  } catch (error) {
    console.error("Errore nell'eliminazione dell'interfaccia:", error)
    globalNotifications.showError("Errore nell'eliminazione dell'interfaccia")
  }
}

// Funzione per ottenere l'icona dello stato
function getStatusIcon(status: string) {
  switch (status) {
    case 'active':
      return 'material-symbols:check-circle'
    case 'inactive':
      return 'material-symbols:cancel'
    case 'pending':
      return 'material-symbols:pending'
    default:
      return 'material-symbols:help-circle'
  }
}

// Funzione per ottenere il colore dello stato
function getStatusColor(status: string) {
  switch (status) {
    case 'active':
      return 'text-success'
    case 'inactive':
      return 'text-error'
    case 'pending':
      return 'text-warning'
    default:
      return 'text-base-content'
  }
}

// Statistiche computate
const interfaceStats = computed(() => {
  const total = interfaces.value.length
  const active = interfaces.value.filter((iface) => iface.status === 'active').length

  return { total, active }
})

function handleInterfaceAdded() {
  showAddForm.value = false
  fetchInterfaces()
}

function handleInterfaceUpdated() {
  editingInterface.value = null
  fetchInterfaces()
}

function handleCancel() {
  showAddForm.value = false
  editingInterface.value = null
}

function showEditForm(iface: Interface) {
  editingInterface.value = iface
  showAddForm.value = false
}

onMounted(async () => {
  await Promise.all([fetchVM(), fetchInterfaces(), fetchNets()])
  isLoading.value = false
})
</script>

<template>
  <div class="space-y-8">
    <!-- Header con breadcrumb -->
    <div class="flex items-center justify-between">
      <div>
        <div class="flex items-center gap-2 text-base-content/70 mb-2">
          <RouterLink to="/vm" class="hover:text-primary transition-colors">
            <IconifyIcon icon="material-symbols:computer" class="text-lg" />
            VM
          </RouterLink>
          <IconifyIcon icon="material-symbols:chevron-right" class="text-sm" />
          <span class="text-base-content">{{ vm?.name || `VM #${vmid}` }}</span>
        </div>
        <h1 class="text-4xl font-bold text-base-content mb-2">
          <IconifyIcon icon="material-symbols:network-node" class="inline mr-3 text-primary" />
          Interfacce di Rete
        </h1>
        <p class="text-base-content/70 text-lg">
          Gestisci le connessioni di rete per {{ vm?.name || `VM #${vmid}` }}
        </p>
      </div>

      <RouterLink to="/vm" class="btn btn-ghost">
        <IconifyIcon icon="material-symbols:arrow-back" class="text-lg" />
        Torna alle VM
      </RouterLink>
    </div>

    <!-- Statistiche -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
      <div class="liquid-glass-card p-6">
        <div class="flex items-center gap-4">
          <div class="btn btn-primary btn-square btn-lg">
            <IconifyIcon icon="material-symbols:network-node" class="text-2xl" />
          </div>
          <div>
            <h3 class="text-2xl font-bold text-base-content">{{ interfaceStats.total }}</h3>
            <p class="text-base-content/70 font-semibold">Interfacce Totali</p>
          </div>
        </div>
      </div>

      <div class="liquid-glass-card p-6">
        <div class="flex items-center gap-4">
          <div class="btn btn-success btn-square btn-lg">
            <IconifyIcon icon="material-symbols:check-circle" class="text-2xl" />
          </div>
          <div>
            <h3 class="text-2xl font-bold text-base-content">{{ interfaceStats.active }}</h3>
            <p class="text-base-content/70 font-semibold">Attive</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Form Aggiunta/Modifica Interfaccia -->
    <div
      class="liquid-glass-card-no-scale bg-gradient-to-br from-primary/5 via-transparent to-info/5"
    >
      <div class="p-6">
        <button
          v-if="!showAddForm && !editingInterface"
          @click="showAddForm = true"
          class="w-full flex items-center justify-center gap-3 text-left"
        >
          <IconifyIcon icon="material-symbols:add-circle" class="text-2xl text-primary" />
          <h2 class="text-xl font-bold text-base-content">Aggiungi Nuova Interfaccia</h2>
        </button>

        <div v-if="showAddForm || editingInterface" class="space-y-4">
          <div class="flex items-center gap-3 mb-4">
            <Icon
              :icon="editingInterface ? 'material-symbols:edit' : 'material-symbols:add-circle'"
              class="text-2xl text-primary"
            />
            <h2 class="text-xl font-bold text-base-content">
              {{ editingInterface ? 'Modifica Interfaccia' : 'Nuova Interfaccia' }}
            </h2>
          </div>

          <InterfaceForm
            v-if="showAddForm"
            :vmid="vmid"
            @interface-added="handleInterfaceAdded"
            @cancel="handleCancel"
          />
          <InterfaceForm
            v-if="editingInterface"
            :vmid="vmid"
            :interface="editingInterface"
            @interface-updated="handleInterfaceUpdated"
            @cancel="handleCancel"
          />
        </div>
      </div>
    </div>

    <!-- Alert informativi -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div
        class="liquid-glass-card-no-scale bg-gradient-to-br from-warning/10 via-transparent to-warning/5 p-6"
      >
        <div class="flex items-start gap-4">
          <IconifyIcon
            icon="material-symbols:warning"
            class="text-2xl text-warning flex-shrink-0 mt-1"
          />
          <div>
            <h3 class="font-bold text-base-content mb-2">Attenzione</h3>
            <p class="text-base-content/80 text-sm leading-relaxed">
              È possibile aggiungere interfacce mentre la VM è in esecuzione. La VM vedrà
              l'interfaccia, ma non sarà configurata automaticamente. Per configurare l'interfaccia
              sarà necessario riavviare la VM.
            </p>
          </div>
        </div>
      </div>

      <div
        class="liquid-glass-card-no-scale bg-gradient-to-br from-info/10 via-transparent to-info/5 p-6"
      >
        <div class="flex items-start gap-4">
          <IconifyIcon icon="material-symbols:info" class="text-2xl text-info flex-shrink-0 mt-1" />
          <div>
            <h3 class="font-bold text-base-content mb-2">Informazioni VLAN</h3>
            <p class="text-base-content/80 text-sm leading-relaxed">
              Il tag VLAN è opzionale (usa 0 se non sai cosa inserire). Serve per separare diverse
              VM a livello 2. Le interfacce con lo stesso tag VLAN possono comunicare tra loro. Il
              gateway è sulla VLAN 0 (untagged).
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Lista Interfacce -->
    <div>
      <h2 class="text-xl font-bold text-base-content mb-4">Interfacce Configurate</h2>

      <div v-if="interfaces.length === 0" class="liquid-glass-card-no-scale p-8 text-center">
        <IconifyIcon
          icon="material-symbols:network-off"
          class="text-6xl text-base-content/30 mb-4"
        />
        <h3 class="text-lg font-semibold text-base-content/70 mb-2">
          Nessuna interfaccia configurata
        </h3>
        <p class="text-base-content/50">Aggiungi la prima interfaccia di rete per questa VM!</p>
      </div>

      <!-- Interface Cards -->
      <div v-else class="space-y-4">
        <div v-for="iface in interfaces" :key="iface.id" class="liquid-glass-card-no-scale p-6">
          <div class="flex items-center justify-between">
            <!-- Informazioni Interfaccia -->
            <div class="flex items-center gap-6">
              <!-- Stato -->
              <div class="flex items-center gap-3">
                <div
                  class="btn btn-square btn-md rounded-xl p-0 m-1 flex-shrink-0"
                  :class="{
                    'btn-success': iface.status === 'active',
                    'btn-error': iface.status === 'inactive',
                    'btn-warning': iface.status === 'pending',
                    'btn-neutral': !['active', 'inactive', 'pending'].includes(iface.status),
                  }"
                >
                  <IconifyIcon :icon="getStatusIcon(iface.status)" class="text-xl" />
                </div>
                <div>
                  <h3 class="font-bold text-lg text-base-content">Interfaccia #{{ iface.id }}</h3>
                  <span
                    class="text-sm font-semibold capitalize"
                    :class="getStatusColor(iface.status)"
                  >
                    {{ iface.status }}
                  </span>
                </div>
              </div>

              <!-- Dettagli -->
              <div class="hidden lg:flex items-center gap-6">
                <div class="text-center">
                  <p class="text-sm text-base-content/50 font-medium">Rete</p>
                  <p class="text-base font-bold text-base-content">
                    {{ netMap.get(iface.vnet_id) || 'N/A' }}
                  </p>
                </div>
                <div class="text-center">
                  <p class="text-sm text-base-content/50 font-medium">VLAN Tag</p>
                  <p class="text-sm font-mono text-base-content">{{ iface.vlan_tag }}</p>
                </div>
                <div class="text-center">
                  <p class="text-sm text-base-content/50 font-medium">IP Address</p>
                  <p class="text-sm font-mono text-base-content">{{ iface.ip_add || 'N/A' }}</p>
                </div>
                <div class="text-center">
                  <p class="text-sm text-base-content/50 font-medium">Gateway</p>
                  <p class="text-sm font-mono text-base-content">{{ iface.gateway || 'N/A' }}</p>
                </div>
              </div>
            </div>

            <!-- Azioni -->
            <div class="flex items-center gap-2">
              <button @click="showEditForm(iface)" class="btn btn-primary btn-sm">
                <IconifyIcon icon="material-symbols:edit" class="text-lg" />
                <span class="hidden sm:inline ml-1">Modifica</span>
              </button>

              <button @click="deleteInterface(iface.id)" class="btn btn-error btn-sm">
                <IconifyIcon icon="material-symbols:delete" class="text-lg" />
                <span class="hidden sm:inline ml-1">Elimina</span>
              </button>
            </div>
          </div>

          <!-- Dettagli Mobile -->
          <div class="lg:hidden mt-4 pt-4 border-t border-white/20">
            <div class="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span class="text-base-content/50">Rete: </span>
                <span class="font-semibold text-base-content">{{
                  netMap.get(iface.vnet_id) || 'N/A'
                }}</span>
              </div>
              <div>
                <span class="text-base-content/50">VLAN Tag: </span>
                <span class="font-mono text-base-content">{{ iface.vlan_tag }}</span>
              </div>
              <div>
                <span class="text-base-content/50">IP: </span>
                <span class="font-mono text-xs text-base-content">{{ iface.ip_add || 'N/A' }}</span>
              </div>
              <div>
                <span class="text-base-content/50">Gateway: </span>
                <span class="font-mono text-xs text-base-content">{{
                  iface.gateway || 'N/A'
                }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
