<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'
import type { SSHKey } from '@/types'

const route = useRoute()
const keys = ref<SSHKey[]>([])
const isLoading = ref(true)
const addingKey = ref(false)
const expandedKey = ref<number | null>(null)
const searchQuery = ref('')
const newKey = ref<{ name: string; key: string }>({ name: '', key: '' })

// Chiavi filtrate per ricerca
const filteredKeys = computed(() => {
  if (!searchQuery.value) return keys.value

  const query = searchQuery.value.toLowerCase()
  return keys.value.filter(
    (key) => key.name.toLowerCase().includes(query) || key.key.toLowerCase().includes(query),
  )
})

// Statistiche delle chiavi SSH
const keyStats = computed(() => ({
  totalKeys: keys.value.length,
  rsaKeys: keys.value.filter((key) => key.key.startsWith('ssh-rsa')).length,
  ed25519Keys: keys.value.filter((key) => key.key.startsWith('ssh-ed25519')).length,
  otherKeys: keys.value.filter(
    (key) => !key.key.startsWith('ssh-rsa') && !key.key.startsWith('ssh-ed25519'),
  ).length,
}))

async function fetchKeys() {
  try {
    isLoading.value = true
    const res = await api.get('/admin/ssh-keys/global')
    keys.value = res.data as SSHKey[]
  } catch (error) {
    console.error('Errore nel caricamento delle chiavi SSH:', error)
    globalNotifications.showError('Errore nel caricamento delle chiavi SSH')
    keys.value = []
  } finally {
    isLoading.value = false
  }
}

async function addKey() {
  try {
    if (!newKey.value.name.trim() || !newKey.value.key.trim()) {
      globalNotifications.showError('Nome e chiave sono obbligatori')
      return
    }

    const res = await api.post('/admin/ssh-keys/global', newKey.value)
    keys.value.push(res.data)
    newKey.value.name = ''
    newKey.value.key = ''
    addingKey.value = false
    globalNotifications.showSuccess('Chiave SSH aggiunta con successo!')
  } catch (error) {
    console.error("Errore nell'aggiunta della chiave SSH:", error)
    globalNotifications.showError("Errore nell'aggiunta della chiave SSH")
  }
}

async function deleteKey(key: SSHKey) {
  if (!confirm(`Sei sicuro di voler eliminare la chiave "${key.name}"?`)) {
    return
  }

  try {
    await api.delete(`/admin/ssh-keys/global/${key.id}`)
    keys.value = keys.value.filter((k) => k.id !== key.id)
    globalNotifications.showSuccess('Chiave SSH eliminata con successo!')
  } catch (error) {
    console.error("Errore nell'eliminazione della chiave SSH:", error)
    globalNotifications.showError("Errore nell'eliminazione della chiave SSH")
  }
}

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    globalNotifications.showSuccess('Chiave copiata negli appunti!')
  } catch (error) {
    console.error('Errore nella copia:', error)
    globalNotifications.showError('Errore nella copia negli appunti')
  }
}

function getKeyType(key: string): string {
  if (key.startsWith('ssh-rsa')) return 'RSA'
  if (key.startsWith('ssh-ed25519')) return 'Ed25519'
  if (key.startsWith('ssh-dss')) return 'DSA'
  if (key.startsWith('ecdsa-sha2')) return 'ECDSA'
  return 'Unknown'
}

function getKeyIcon(key: string): string {
  const type = getKeyType(key)
  switch (type) {
    case 'RSA':
      return 'material-symbols:key'
    case 'Ed25519':
      return 'material-symbols:enhanced-encryption'
    case 'ECDSA':
      return 'material-symbols:security'
    default:
      return 'material-symbols:vpn-key'
  }
}

function getKeyColor(key: string): string {
  const type = getKeyType(key)
  switch (type) {
    case 'RSA':
      return 'text-blue-500'
    case 'Ed25519':
      return 'text-green-500'
    case 'ECDSA':
      return 'text-purple-500'
    default:
      return 'text-gray-500'
  }
}

function truncateKey(key: string, length: number = 50): string {
  return key.length > length ? key.substring(0, length) + '...' : key
}

onMounted(() => {
  fetchKeys()

  // Controlla se deve aprire automaticamente il form di aggiunta
  if (route.query.add === 'true') {
    addingKey.value = true
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
        <div class="flex items-center gap-4 mb-4">
          <div
            class="w-12 h-12 rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center shadow-lg"
          >
            <IconifyIcon icon="material-symbols:vpn-key" class="text-2xl text-white" />
          </div>
          <div>
            <h1 class="text-3xl font-bold text-base-content">Chiavi SSH Globali</h1>
            <p class="text-base-content/70">Gestione delle chiavi SSH condivise del sistema</p>
          </div>
        </div>
      </div>

      <!-- Statistiche -->
      <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6 px-2">
        <div
          class="stat bg-gradient-to-br from-primary/10 to-primary/20 border border-primary/20 rounded-xl"
        >
          <div class="stat-figure text-primary">
            <IconifyIcon icon="material-symbols:vpn-key" class="text-3xl" />
          </div>
          <div class="stat-title text-primary/70">Totali</div>
          <div class="stat-value text-2xl text-primary">{{ keyStats.totalKeys }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/20 rounded-xl"
        >
          <div class="stat-figure text-blue-500">
            <IconifyIcon icon="material-symbols:key" class="text-3xl" />
          </div>
          <div class="stat-title text-blue-500/70">RSA</div>
          <div class="stat-value text-2xl text-blue-500">{{ keyStats.rsaKeys }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-green-500/10 to-green-600/10 border border-green-500/20 rounded-xl"
        >
          <div class="stat-figure text-green-500">
            <IconifyIcon icon="material-symbols:enhanced-encryption" class="text-3xl" />
          </div>
          <div class="stat-title text-green-500/70">Ed25519</div>
          <div class="stat-value text-2xl text-green-500">{{ keyStats.ed25519Keys }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-purple-500/10 to-purple-600/10 border border-purple-500/20 rounded-xl"
        >
          <div class="stat-figure text-purple-500">
            <IconifyIcon icon="material-symbols:security" class="text-3xl" />
          </div>
          <div class="stat-title text-purple-500/70">Altri</div>
          <div class="stat-value text-2xl text-purple-500">{{ keyStats.otherKeys }}</div>
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
            placeholder="Cerca per nome o tipo di chiave..."
            class="input input-bordered flex-1"
          />
        </div>

        <!-- Pulsanti azione -->
        <div class="flex gap-2 shrink-0">
          <!-- Pulsante aggiungi chiave -->
          <button @click="addingKey = true" v-show="!addingKey" class="btn btn-primary gap-2 h-12">
            <IconifyIcon icon="material-symbols:add" />
            Nuova Chiave
          </button>

          <!-- Pulsante annulla -->
          <button @click="addingKey = false" v-show="addingKey" class="btn btn-error gap-2 h-12">
            <IconifyIcon icon="material-symbols:cancel" />
            Annulla
          </button>
        </div>
      </div>

      <!-- Form aggiunta chiave -->
      <div v-if="addingKey" class="mb-6 px-2">
        <div
          class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl p-6 shadow-lg"
        >
          <div class="flex items-center gap-3 mb-6">
            <div class="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
              <IconifyIcon icon="material-symbols:add" class="text-xl text-primary" />
            </div>
            <div>
              <h3 class="font-bold text-xl">Aggiungi Nuova Chiave SSH</h3>
              <p class="text-sm text-base-content/70">
                Inserisci i dettagli della nuova chiave SSH globale
              </p>
            </div>
          </div>

          <form @submit.prevent="addKey" class="space-y-4">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium text-base-content">Nome della Chiave</span>
                </label>
                <input
                  v-model="newKey.name"
                  type="text"
                  placeholder="La mia chiave SSH"
                  class="input input-bordered w-full"
                  required
                />
              </div>
            </div>

            <div class="form-control">
              <label class="label">
                <span class="label-text font-medium text-base-content">Chiave SSH</span>
                <span class="label-text-alt text-base-content/60"
                  >Incolla la chiave pubblica SSH</span
                >
              </label>
              <textarea
                v-model="newKey.key"
                placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC..."
                class="textarea textarea-bordered w-full font-mono text-sm"
                rows="4"
                required
              ></textarea>
            </div>

            <div class="flex justify-end gap-3 pt-4 border-t border-base-300">
              <button type="button" @click="addingKey = false" class="btn btn-ghost">
                <IconifyIcon icon="material-symbols:close" />
                Annulla
              </button>
              <button type="submit" class="btn btn-primary">
                <IconifyIcon icon="material-symbols:add" />
                Aggiungi Chiave
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Lista chiavi SSH -->
      <div class="px-2">
        <div
          class="bg-base-100/80 backdrop-blur-sm border border-base-300/50 rounded-2xl shadow-lg overflow-hidden"
        >
          <!-- Loading state -->
          <div v-if="isLoading" class="p-8 text-center">
            <span class="loading loading-spinner loading-lg text-primary"></span>
            <p class="mt-2 text-base-content/70">Caricamento chiavi SSH...</p>
          </div>

          <!-- Tabella chiavi -->
          <div v-else-if="filteredKeys.length > 0" class="overflow-x-auto">
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
                      <IconifyIcon icon="material-symbols:vpn-key" class="text-sm" />
                      Nome & Tipo
                    </div>
                  </th>
                  <th class="bg-base-200/50">
                    <div class="flex items-center gap-2">
                      <IconifyIcon icon="material-symbols:fingerprint" class="text-sm" />
                      Chiave
                    </div>
                  </th>
                  <th class="bg-base-200/50 text-center">Azioni</th>
                </tr>
              </thead>
              <tbody>
                <template v-for="key in filteredKeys" :key="key.id">
                  <tr class="hover">
                    <td class="font-mono text-sm">{{ key.id }}</td>
                    <td>
                      <div class="flex items-center gap-3">
                        <div
                          class="w-10 h-10 rounded-full flex items-center justify-center"
                          :class="
                            getKeyColor(key.key).replace('text-', 'bg-').replace('-500', '-100')
                          "
                        >
                          <Icon
                            :icon="getKeyIcon(key.key)"
                            :class="getKeyColor(key.key)"
                            class="text-xl"
                          />
                        </div>
                        <div>
                          <div class="font-bold">{{ key.name }}</div>
                          <div class="text-sm text-base-content/70">{{ getKeyType(key.key) }}</div>
                        </div>
                      </div>
                    </td>
                    <td class="font-mono text-sm text-base-content/70">
                      {{ truncateKey(key.key) }}
                    </td>
                    <td>
                      <div class="flex gap-2 justify-center">
                        <button
                          @click="expandedKey = expandedKey === key.id ? null : key.id"
                          class="btn btn-ghost btn-sm gap-2 hover:btn-info"
                          :class="{ 'btn-info': expandedKey === key.id }"
                        >
                          <Icon
                            :icon="
                              expandedKey === key.id
                                ? 'material-symbols:expand-less'
                                : 'material-symbols:visibility'
                            "
                          />
                          {{ expandedKey === key.id ? 'Chiudi' : 'Visualizza' }}
                        </button>

                        <button
                          @click="deleteKey(key)"
                          class="btn btn-ghost btn-sm gap-2 hover:btn-error"
                        >
                          <IconifyIcon icon="material-symbols:delete" />
                          Elimina
                        </button>
                      </div>
                    </td>
                  </tr>

                  <!-- Riga espandibile per visualizzazione chiave completa -->
                  <tr v-if="expandedKey === key.id" class="bg-base-200/30">
                    <td colspan="4" class="p-0">
                      <div class="p-6">
                        <div class="bg-base-100 rounded-xl p-6 shadow-sm border border-base-300">
                          <div class="flex items-center gap-3 mb-6">
                            <div
                              class="w-10 h-10 rounded-full flex items-center justify-center"
                              :class="
                                getKeyColor(key.key).replace('text-', 'bg-').replace('-500', '-100')
                              "
                            >
                              <Icon
                                :icon="getKeyIcon(key.key)"
                                :class="getKeyColor(key.key)"
                                class="text-xl"
                              />
                            </div>
                            <div>
                              <h3 class="text-lg font-bold">{{ key.name }}</h3>
                              <p class="text-sm text-base-content/70">
                                Chiave SSH {{ getKeyType(key.key) }}
                              </p>
                            </div>
                          </div>

                          <!-- Visualizzazione chiave completa -->
                          <div class="space-y-4">
                            <div class="form-control">
                              <label class="label">
                                <span class="label-text font-medium text-base-content"
                                  >Chiave SSH Completa</span
                                >
                                <button
                                  @click="copyToClipboard(key.key)"
                                  class="btn btn-ghost btn-xs gap-1"
                                  title="Copia negli appunti"
                                >
                                  <IconifyIcon icon="material-symbols:content-copy" />
                                  Copia
                                </button>
                              </label>
                              <textarea
                                :value="key.key"
                                readonly
                                class="textarea textarea-bordered w-full font-mono text-sm bg-base-200/50"
                                rows="6"
                              ></textarea>
                            </div>
                          </div>

                          <!-- Pulsante chiudi -->
                          <div class="flex justify-end mt-6 pt-4 border-t border-base-300">
                            <button @click="expandedKey = null" class="btn btn-ghost">
                              <IconifyIcon icon="material-symbols:close" />
                              Chiudi
                            </button>
                          </div>
                        </div>
                      </div>
                    </td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>

          <!-- Stato vuoto -->
          <div v-else-if="keys.length === 0" class="p-8 text-center">
            <IconifyIcon
              icon="material-symbols:vpn-key-off"
              class="text-6xl text-base-content/30 mb-4"
            />
            <p class="text-lg font-medium text-base-content/70 mb-2">
              Nessuna chiave SSH configurata
            </p>
            <p class="text-base-content/50 mb-4">
              Aggiungi la prima chiave SSH globale per iniziare
            </p>
            <button @click="addingKey = true" class="btn btn-primary gap-2">
              <IconifyIcon icon="material-symbols:add" />
              Aggiungi Prima Chiave
            </button>
          </div>

          <!-- Stato vuoto per ricerca -->
          <div v-else class="p-8 text-center">
            <IconifyIcon
              icon="material-symbols:search-off"
              class="text-6xl text-base-content/30 mb-4"
            />
            <p class="text-base-content/70">Nessuna chiave trovata per "{{ searchQuery }}"</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
