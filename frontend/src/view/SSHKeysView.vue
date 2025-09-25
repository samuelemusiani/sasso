<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import type { SSHKey } from '@/types'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const keys = ref<SSHKey[]>([])
const name = ref('')
const key = ref('')
const isLoading = ref(true)

// Statistiche computate
const keyStats = computed(() => {
  const total = keys.value.length
  const recent = Math.min(keys.value.length, 5)
  
  return { total, recent }
})

async function fetchSSHKeys() {
  try {
    isLoading.value = true
    const response = await api.get('/ssh-keys')
    keys.value = response.data
  } catch (error) {
    console.error('Errore nel recuperare le chiavi SSH:', error)
  } finally {
    isLoading.value = false
  }
}

async function addSSHKey() {
  if (!name.value.trim() || !key.value.trim()) {
    globalNotifications.showWarning('Inserisci sia il nome che la chiave SSH')
    return
  }
  
  try {
    await api.post('/ssh-keys', {
      name: name.value.trim(),
      key: key.value.trim(),
    })
    
    // Reset form
    name.value = ''
    key.value = ''
    
    await fetchSSHKeys()
    globalNotifications.showSuccess('Chiave SSH aggiunta con successo!')
  } catch (error) {
    console.error('Errore nell\'aggiunta della chiave SSH:', error)
    globalNotifications.showError('Errore nell\'aggiunta della chiave SSH')
  }
}

async function deleteSSHKey(id: number) {
  if (!confirm('Sei sicuro di voler eliminare questa chiave SSH?')) {
    return
  }

  try {
    await api.delete(`/ssh-keys/${id}`)
    console.log(`Chiave SSH ${id} eliminata con successo`)
    await fetchSSHKeys()
    globalNotifications.showSuccess('Chiave SSH eliminata con successo!')
  } catch (error) {
    console.error(`Errore nell'eliminazione della chiave SSH ${id}:`, error)
    globalNotifications.showError('Errore nell\'eliminazione della chiave SSH')
  }
}

onMounted(() => {
  fetchSSHKeys()
})
</script>

<template>
  <div class="space-y-8">
    
    <!-- Header -->
    <div class="text-center">
      <h1 class="text-4xl font-bold text-base-content mb-2">
        <Icon icon="material-symbols:key" class="inline mr-3 text-primary" />
        Gestione Chiavi SSH
      </h1>
      <p class="text-base-content/70 text-lg">Gestisci le tue chiavi SSH per l'accesso sicuro ai server</p>
    </div>

    <!-- Statistiche -->
    <div class="flex justify-center">
      <div class="liquid-glass-card p-6">
        <div class="flex items-center gap-4">
          <div class="btn btn-primary btn-square btn-lg">
            <Icon icon="material-symbols:key" class="text-2xl" />
          </div>
          <div>
            <h3 class="text-2xl font-bold text-base-content">{{ keyStats.total }}</h3>
            <p class="text-base-content/70 font-semibold">Chiavi Totali</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Form per aggiungere nuova chiave SSH -->
    <div class="liquid-glass-card">
      <div class="card-body p-6">
        <h2 class="card-title text-base-content mb-6 flex items-center gap-3">
          <Icon icon="material-symbols:add-circle" class="text-primary text-2xl" />
          Aggiungi Nuova Chiave SSH
        </h2>
        
        <div class="flex flex-col lg:flex-row gap-8 mb-6">
          <div class="form-control lg:w-1/4">
            <label class="label">
              <span class="label-text text-base-content font-semibold">Nome della Chiave</span>
            </label>
            <input 
              type="text" 
              v-model="name" 
              placeholder="Inserisci il nome della chiave..."
              class="input input-bordered bg-base-100/10 border-white/30 focus:border-primary/50 text-base-content placeholder:text-base-content/50 w-full"
            />
          </div>
          
          <!-- Spazio vuoto per separazione -->
          <div class="hidden lg:block lg:w-8"></div>
          
          <div class="form-control lg:flex-1">
            <label class="label">
              <span class="label-text text-base-content font-semibold">Chiave Pubblica SSH</span>
            </label>
            <textarea 
              v-model="key" 
              rows="3"
              placeholder="Incolla qui la tua chiave pubblica SSH..."
              class="textarea textarea-bordered bg-base-100/10 border-white/30 focus:border-primary/50 text-base-content placeholder:text-base-content/50 resize-none w-full"
            ></textarea>
          </div>
        </div>
        
        <div class="card-actions justify-end">
          <button 
            @click="addSSHKey()"
            class="btn btn-primary gap-2"
          >
            <Icon icon="material-symbols:add" class="text-lg" />
            Aggiungi Chiave SSH
          </button>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center py-12">
      <span class="loading loading-spinner loading-lg text-primary"></span>
    </div>

    <!-- Empty State -->
    <div v-else-if="keys.length === 0" class="text-center py-12">
      <div class="liquid-glass-card p-8">
        <Icon icon="material-symbols:key-off" class="text-6xl text-base-content/30 mx-auto mb-4" />
        <h3 class="text-xl font-semibold text-base-content mb-2">Nessuna Chiave SSH Trovata</h3>
        <p class="text-base-content/70">Aggiungi la tua prima chiave SSH per iniziare con l'accesso sicuro ai server.</p>
      </div>
    </div>

    <!-- Lista Chiavi SSH -->
    <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div 
        v-for="sshKey in keys" 
        :key="sshKey.id"
        class="liquid-glass-card hover:scale-[1.02] transition-transform duration-200"
      >
        <div class="card-body p-6">
          <div class="flex items-start justify-between mb-4">
            <div class="flex-1">
              <h3 class="card-title text-base-content text-lg mb-1">{{ sshKey.name }}</h3>
              <p class="text-sm text-base-content/60">ID: {{ sshKey.id }}</p>
            </div>
            <div class="flex items-center gap-2">
              <button
                @click="deleteSSHKey(sshKey.id)"
                class="btn btn-error btn-sm btn-square"
                title="Elimina Chiave SSH"
              >
                <Icon icon="material-symbols:delete" class="text-lg" />
              </button>
            </div>
          </div>
          
          <div class="bg-base-200/50 rounded-lg p-3 border border-base-300/50">
            <p class="text-xs text-base-content/60 mb-1 font-semibold">Chiave Pubblica:</p>
            <p class="text-sm font-mono text-base-content break-all leading-relaxed">
              {{ sshKey.key.length > 80 ? sshKey.key.substring(0, 80) + '...' : sshKey.key }}
            </p>
          </div>
          
          <div class="mt-4 flex items-center text-xs text-base-content/60">
            <span class="flex items-center gap-1">
              <Icon icon="material-symbols:security" class="text-sm" />
              {{ sshKey.key.split(' ')[0] || 'Tipo Sconosciuto' }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
