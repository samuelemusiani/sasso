<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { api } from '@/lib/api'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'
import type { Realm, LDAPRealm } from '@/types'
import RealmsMultiplexer from '@/components/realms/RealmsMultiplexer.vue'
import LDAPForm from '@/components/realms/LDAPForm.vue'

const route = useRoute()

const realms = ref<Realm[]>([])
const isLoading = ref(true)
const addingRealm = ref(false)
const addingType = ref('ldap')
const searchQuery = ref('')
const expandedRealm = ref<number | null>(null)
const editingRealm = ref<Realm | null>(null)
const isUpdating = ref(false)

// Realms filtrati per ricerca
const filteredRealms = computed(() => {
  if (!searchQuery.value) return realms.value

  const query = searchQuery.value.toLowerCase()
  return realms.value.filter(
    (realm) =>
      realm.name.toLowerCase().includes(query) ||
      realm.description.toLowerCase().includes(query) ||
      realm.type.toLowerCase().includes(query),
  )
})

// Statistiche realms
const realmStats = computed(() => {
  const total = realms.value.length
  const ldapRealms = realms.value.filter((r) => r.type === 'ldap').length
  const localRealms = realms.value.filter((r) => r.type === 'local').length

  return { total, ldapRealms, localRealms }
})

async function fetchRealms() {
  try {
    isLoading.value = true
    const response = await api.get('/admin/realms')
    realms.value = response.data as Realm[]
  } catch (error) {
    console.error('Errore nel caricamento realms:', error)
    globalNotifications.showError('Errore nel caricamento dei realms')
  } finally {
    isLoading.value = false
  }
}

function realmAdded() {
  addingRealm.value = false
  fetchRealms()
  globalNotifications.showSuccess('Realm aggiunto con successo!')
}

async function deleteRealm(realm: Realm) {
  if (!confirm(`Sei sicuro di voler eliminare il realm "${realm.name}"?`)) {
    return
  }

  try {
    await api.delete(`/admin/realms/${realm.id}`)
    await fetchRealms()
    globalNotifications.showSuccess(`Realm "${realm.name}" eliminato con successo!`)
  } catch (error) {
    console.error(`Errore nell'eliminazione del realm ${realm.id}:`, error)
    globalNotifications.showError(`Errore nell'eliminazione del realm "${realm.name}"`)
  }
}

function getRealmIcon(type: string) {
  switch (type) {
    case 'ldap':
      return 'material-symbols:account-tree'
    case 'local':
      return 'material-symbols:computer'
    case 'oauth':
      return 'material-symbols:key'
    default:
      return 'material-symbols:domain'
  }
}

function getRealmColor(type: string) {
  switch (type) {
    case 'ldap':
      return 'text-blue-500'
    case 'local':
      return 'text-green-500'
    case 'oauth':
      return 'text-purple-500'
    default:
      return 'text-orange-500'
  }
}

function getRealmBadgeClass(type: string) {
  switch (type) {
    case 'ldap':
      return 'badge-primary'
    case 'local':
      return 'badge-success'
    case 'oauth':
      return 'badge-secondary'
    default:
      return 'badge-warning'
  }
}

async function toggleRealmEdit(realm: Realm) {
  if (expandedRealm.value === realm.id) {
    expandedRealm.value = null
    editingRealm.value = null
  } else {
    expandedRealm.value = realm.id

    // Recupera i dettagli completi del realm per l'editing
    if (realm.type === 'ldap') {
      try {
        const response = await api.get(`/admin/realms/${realm.id}`)
        editingRealm.value = response.data as LDAPRealm
      } catch (error) {
        console.error('Errore nel recupero dettagli realm:', error)
        globalNotifications.showError('Errore nel caricamento dei dettagli del realm')
        expandedRealm.value = null
      }
    } else {
      editingRealm.value = { ...realm }
    }
  }
}

async function updateRealm(realmId: number) {
  try {
    isUpdating.value = true

    if (!editingRealm.value) {
      globalNotifications.showError('Nessun realm in modifica')
      return
    }

    const realmData = editingRealm.value

    await api.put(`/admin/realms/${realmId}`, {
      name: realmData.name,
      description: realmData.description,
      // Altri campi specifici del realm potrebbero essere aggiunti qui
    })

    // Aggiorna i dati locali
    const realmIndex = realms.value.findIndex((r) => r.id === realmId)
    if (realmIndex !== -1) {
      realms.value[realmIndex] = { ...realmData }
    }

    expandedRealm.value = null
    editingRealm.value = null

    globalNotifications.showSuccess('Realm aggiornato con successo!')
  } catch (error) {
    console.error("Errore nell'aggiornamento del realm:", error)
    globalNotifications.showError("Errore nell'aggiornamento del realm")
  } finally {
    isUpdating.value = false
  }
}

function handleRealmUpdate() {
  // Questa funzione viene chiamata quando il form LDAP ha completato l'aggiornamento
  expandedRealm.value = null
  editingRealm.value = null
  fetchRealms() // Ricarica i dati
  globalNotifications.showSuccess('Realm aggiornato con successo!')
}

onMounted(() => {
  fetchRealms()

  // Controlla se deve aprire automaticamente il form di creazione
  if (route.query.add === 'true') {
    addingRealm.value = true
  }
})
</script>

<template>
  <div class="h-full overflow-auto">
    <!-- Header con breadcrumb -->
    <div class="mb-6 px-2">
      <div class="flex items-center gap-2 mb-2">
        <RouterLink to="/admin" class="btn btn-ghost btn-sm gap-2">
          <IconifyIcon icon="material-symbols:arrow-back" />
          Admin Panel
        </RouterLink>
        <span class="text-base-content/50">/</span>
        <span class="text-base-content font-medium">Domini Autenticazione</span>
      </div>

      <div class="flex items-center gap-3 mb-4">
        <div class="btn btn-square btn-lg rounded-xl btn-primary p-0 flex-shrink-0">
          <IconifyIcon icon="material-symbols:domain-verification" class="text-2xl" />
        </div>
        <div>
          <h1 class="text-3xl font-bold text-base-content">Domini Autenticazione</h1>
          <p class="text-base-content/70">Gestisci LDAP, OAuth e altri metodi di autenticazione</p>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="flex justify-center items-center h-64">
      <div class="loading loading-spinner loading-lg"></div>
      <span class="ml-4 text-lg">Caricamento realms...</span>
    </div>

    <div v-else>
      <!-- Statistiche realms -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6 px-2">
        <div
          class="stat bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/20 rounded-xl"
        >
          <div class="stat-figure text-blue-500">
            <IconifyIcon icon="material-symbols:domain-verification" class="text-3xl" />
          </div>
          <div class="stat-title text-blue-500/70">Totale Realms</div>
          <div class="stat-value text-2xl text-blue-500">{{ realmStats.total }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-indigo-500/10 to-indigo-600/10 border border-indigo-500/20 rounded-xl"
        >
          <div class="stat-figure text-indigo-500">
            <IconifyIcon icon="material-symbols:account-tree" class="text-3xl" />
          </div>
          <div class="stat-title text-indigo-500/70">LDAP</div>
          <div class="stat-value text-2xl text-indigo-500">{{ realmStats.ldapRealms }}</div>
        </div>

        <div
          class="stat bg-gradient-to-br from-green-500/10 to-green-600/10 border border-green-500/20 rounded-xl"
        >
          <div class="stat-figure text-green-500">
            <IconifyIcon icon="material-symbols:computer" class="text-3xl" />
          </div>
          <div class="stat-title text-green-500/70">Locali</div>
          <div class="stat-value text-2xl text-green-500">{{ realmStats.localRealms }}</div>
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
            placeholder="Cerca per nome, descrizione o tipo..."
            class="input input-bordered flex-1"
          />
        </div>

        <!-- Pulsanti azione -->
        <div class="flex gap-2 shrink-0">
          <!-- Pulsante aggiungi realm -->
          <button
            @click="addingRealm = true"
            v-show="!addingRealm"
            class="btn btn-primary gap-2 h-12"
          >
            <IconifyIcon icon="material-symbols:add" />
            Nuovo Realm
          </button>

          <!-- Pulsante annulla -->
          <button
            @click="addingRealm = false"
            v-show="addingRealm"
            class="btn btn-error gap-2 h-12"
          >
            <IconifyIcon icon="material-symbols:cancel" />
            Annulla
          </button>
        </div>
      </div>

      <!-- Form aggiunta realm -->
      <div v-if="addingRealm" class="mb-6 px-2">
        <div class="card shadow-xl bg-base-100 border border-base-300">
          <div class="card-body">
            <div class="flex items-center gap-3 mb-4">
              <IconifyIcon icon="material-symbols:add-circle" class="text-3xl text-primary" />
              <div>
                <h3 class="font-bold text-xl">Aggiungi Nuovo Realm</h3>
                <p class="text-sm text-base-content/70">
                  Configura un nuovo metodo di autenticazione
                </p>
              </div>
            </div>

            <RealmsMultiplexer :adding="addingRealm" :type="addingType" @realm-added="realmAdded" />
          </div>
        </div>
      </div>

      <!-- Tabella realms moderna -->
      <div v-show="!addingRealm" class="px-2">
        <div class="card shadow-xl bg-base-100 border border-base-300">
          <div class="card-body p-0">
            <div class="overflow-x-auto">
              <table class="table table-zebra w-full">
                <thead>
                  <tr class="bg-base-200">
                    <th class="font-bold">
                      <IconifyIcon icon="material-symbols:tag" class="inline mr-2" />
                      ID
                    </th>
                    <th class="font-bold">
                      <IconifyIcon icon="material-symbols:domain" class="inline mr-2" />
                      Nome
                    </th>
                    <th class="font-bold">
                      <IconifyIcon icon="material-symbols:description" class="inline mr-2" />
                      Descrizione
                    </th>
                    <th class="font-bold">
                      <IconifyIcon icon="material-symbols:category" class="inline mr-2" />
                      Tipo
                    </th>
                    <th class="font-bold">
                      <IconifyIcon icon="material-symbols:settings" class="inline mr-2" />
                      Azioni
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <template v-for="realm in filteredRealms" :key="realm.id">
                    <tr class="hover">
                      <td class="font-mono text-sm">{{ realm.id }}</td>
                      <td>
                        <div class="flex items-center gap-3">
                          <div
                            class="w-10 h-10 rounded-full flex items-center justify-center"
                            :class="
                              getRealmColor(realm.type)
                                .replace('text-', 'bg-')
                                .replace('-500', '-100')
                            "
                          >
                            <Icon
                              :icon="getRealmIcon(realm.type)"
                              :class="getRealmColor(realm.type)"
                              class="text-xl"
                            />
                          </div>
                          <div>
                            <div class="font-bold">{{ realm.name }}</div>
                          </div>
                        </div>
                      </td>
                      <td class="text-sm text-base-content/70">{{ realm.description }}</td>
                      <td>
                        <div class="flex items-center gap-2">
                          <Icon
                            :icon="getRealmIcon(realm.type)"
                            :class="getRealmColor(realm.type)"
                          />
                          <span class="badge badge-sm" :class="getRealmBadgeClass(realm.type)">
                            {{ realm.type.toUpperCase() }}
                          </span>
                        </div>
                      </td>
                      <td>
                        <div class="flex gap-2" v-if="realm.type !== 'local'">
                          <button
                            @click="toggleRealmEdit(realm)"
                            class="btn btn-ghost btn-sm gap-2 hover:btn-primary"
                            :class="{ 'btn-primary': expandedRealm === realm.id }"
                          >
                            <Icon
                              :icon="
                                expandedRealm === realm.id
                                  ? 'material-symbols:expand-less'
                                  : 'material-symbols:edit'
                              "
                            />
                            {{ expandedRealm === realm.id ? 'Chiudi' : 'Modifica' }}
                          </button>

                          <button
                            @click="deleteRealm(realm)"
                            class="btn btn-ghost btn-sm gap-2 hover:btn-error"
                          >
                            <IconifyIcon icon="material-symbols:delete" />
                            Elimina
                          </button>
                        </div>
                        <div v-else class="text-sm text-base-content/50 italic">
                          Realm di sistema
                        </div>
                      </td>
                    </tr>

                    <!-- Riga espandibile per editing realm -->
                    <tr
                      v-if="expandedRealm === realm.id && realm.type !== 'local'"
                      class="bg-base-200/30"
                    >
                      <td colspan="5" class="p-0">
                        <div class="p-6">
                          <div class="bg-base-100 rounded-xl p-6 shadow-sm border border-base-300">
                            <div class="flex items-center gap-3 mb-6">
                              <div
                                class="w-10 h-10 rounded-full flex items-center justify-center"
                                :class="
                                  getRealmColor(realm.type)
                                    .replace('text-', 'bg-')
                                    .replace('-500', '-100')
                                "
                              >
                                <Icon
                                  :icon="getRealmIcon(realm.type)"
                                  :class="getRealmColor(realm.type)"
                                  class="text-xl"
                                />
                              </div>
                              <div>
                                <h3 class="text-lg font-bold">Modifica {{ realm.name }}</h3>
                                <p class="text-sm text-base-content/70">
                                  Configura i parametri del realm {{ realm.type.toUpperCase() }}
                                </p>
                              </div>
                            </div>

                            <!-- Form per editing realm -->
                            <div class="mt-4">
                              <LDAPForm
                                v-if="realm.type === 'ldap' && editingRealm"
                                :realm="editingRealm as LDAPRealm"
                                @realm-added="handleRealmUpdate"
                              />
                              <div
                                v-else-if="realm.type !== 'ldap'"
                                class="text-center py-8 text-base-content/60"
                              >
                                <IconifyIcon
                                  icon="material-symbols:construction"
                                  class="text-4xl mb-2"
                                />
                                <p>
                                  Editing per realm di tipo {{ realm.type.toUpperCase() }} non
                                  ancora supportato
                                </p>
                              </div>
                            </div>

                            <!-- Pulsanti azione -->
                            <div class="flex justify-end gap-3 mt-6 pt-4 border-t border-base-300">
                              <button
                                @click="((expandedRealm = null), (editingRealm = null))"
                                class="btn btn-ghost"
                              >
                                <IconifyIcon icon="material-symbols:close" />
                                Annulla
                              </button>
                              <button
                                @click="updateRealm(realm.id)"
                                class="btn btn-primary"
                                :disabled="isUpdating"
                              >
                                <span
                                  v-if="isUpdating"
                                  class="loading loading-spinner loading-sm"
                                ></span>
                                <IconifyIcon v-else icon="material-symbols:save" />
                                {{ isUpdating ? 'Salvataggio...' : 'Salva modifiche' }}
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

            <!-- Stato vuoto per ricerca -->
            <div v-if="filteredRealms.length === 0 && searchQuery" class="p-8 text-center">
              <IconifyIcon
                icon="material-symbols:search-off"
                class="text-6xl text-base-content/30 mb-4"
              />
              <p class="text-base-content/70">Nessun realm trovato per "{{ searchQuery }}"</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Router view per pagine annidate -->
    <router-view class="mt-6 px-2" />
  </div>
</template>
