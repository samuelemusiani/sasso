<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import { useRouter } from 'vue-router'
import { login as _login } from '@/lib/api'
import type { Realm } from '@/types'
import { Icon } from '@iconify/vue'
import { globalNotifications } from '@/lib/notifications'

const router = useRouter()

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const selectedRealm = ref('Local')
const isLoading = ref(false)
const animationComplete = ref(false)

const realms = ref<Realm[]>([
  { id: 1, name: 'Local', description: 'Local Authentication', type: 'local' },
  { id: 2, name: 'LDAP', description: 'LDAP Authentication', type: 'ldap' }
])

async function fetchRealms() {
  try {
    const response = await api.get('/login/realms')
    const apiRealms = response.data as Realm[]
    
    // Filtra i realm dall'API per rimuovere eventuali duplicati Local
    const nonLocalRealms = apiRealms.filter(realm => realm.type !== 'local')
    
    realms.value = [
      { id: 0, name: 'Local', description: 'Local Authentication', type: 'local' },
      ...nonLocalRealms
    ]
  } catch (error) {
    console.error('Errore nel recuperare i realm:', error)
    // Fallback con realm locali
    realms.value = [
      { id: 1, name: 'Local', description: 'Local Authentication', type: 'local' },
      { id: 2, name: 'LDAP', description: 'LDAP Authentication', type: 'ldap' }
    ]
  }
}

async function login() {
  if (!username.value.trim() || !password.value.trim()) {
    globalNotifications.showError('Campi obbligatori', 'Inserisci username e password')
    return
  }

  try {
    isLoading.value = true
    const realmData = realms.value.find((r) => r.name === selectedRealm.value)
    if (!realmData) {
      globalNotifications.showError('Errore configurazione', 'Realm selezionato non valido')
      return
    }
    
    await _login(username.value.trim(), password.value, realmData.id)
    globalNotifications.showSuccess('Login effettuato', 'Benvenuto nel sistema SASSO!')
    router.push('/')
    
  } catch (error) {
    console.error('Login fallito:', error)
    globalNotifications.showError('Login fallito', 'Verifica le credenziali e riprova')
  } finally {
    isLoading.value = false
  }
}

// Gestione Enter per il login
function handleKeyPress(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    login()
  }
}

onMounted(async () => {
  await fetchRealms()
  
  // Animazione immediata
  setTimeout(() => {
    animationComplete.value = true
  }, 100)
})
</script>

<template>
  <div class="min-h-screen bg-base-300 flex items-center justify-center p-4 relative overflow-hidden">
    <!-- Pattern di sfondo -->
    <div class="absolute inset-0 opacity-10">
      <div class="absolute inset-0" style="background-image: url('/pattern-sasso.png'); background-size: 400px; background-repeat: repeat;"></div>
    </div>
    <!-- Container principale con liquid glass -->
    <div class="w-full max-w-md">
      
      <!-- Logo animato -->
      <div class="text-center mb-8">
        <div 
          class="logo-container transform transition-all duration-300 ease-out"
          :class="animationComplete ? 'translate-y-0 scale-100 opacity-100' : 'translate-y-4 scale-95 opacity-0'"
        >
          <!-- Logo -->
          <div class="w-32 h-32 mx-auto mb-4 flex items-center justify-center">
            <img src="/logo-sasso.png" alt="SASSO Logo" class="w-full h-full object-contain" />
          </div>
          
          <!-- Scritta che appare dopo l'animazione -->
          <div 
            class="transition-all duration-300 ease-out"
            :class="animationComplete ? 'opacity-100 transform translate-y-0' : 'opacity-0 transform translate-y-2'"
          >
            <img src="/logo_scritta.png" alt="SASSO" class="h-12 mx-auto mb-2" />
          </div>
        </div>
      </div>

      <!-- Form di login con liquid glass -->
      <div 
        class="liquid-glass-card transition-all duration-300 ease-out"
        :class="animationComplete ? 'opacity-100 transform translate-y-0' : 'opacity-0 transform translate-y-4'"
      >
        <div class="card-body p-8">
          <h2 class="text-xl font-semibold text-base-content mb-6 text-center">Accedi al Sistema</h2>
          
          <!-- Campo Username -->
          <div class="form-control mb-4">
            <label class="label">
              <span class="label-text text-base-content font-medium">Username</span>
            </label>
            <div class="relative">
              <Icon icon="material-symbols:person" class="absolute left-3 top-1/2 transform -translate-y-1/2 text-base-content/50" />
              <input 
                v-model="username"
                type="text" 
                placeholder="Inserisci il tuo username"
                class="input input-bordered w-full pl-10 bg-base-100/50 border-white/30 focus:border-primary/50 text-base-content placeholder:text-base-content/50"
                @keypress="handleKeyPress"
                :disabled="isLoading"
              />
            </div>
          </div>

          <!-- Campo Password -->
          <div class="form-control mb-4">
            <label class="label">
              <span class="label-text text-base-content font-medium">Password</span>
            </label>
            <div class="relative">
              <Icon icon="material-symbols:lock" class="absolute left-3 top-1/2 transform -translate-y-1/2 text-base-content/50" />
              <input 
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                placeholder="Inserisci la tua password"
                class="input input-bordered w-full pl-10 pr-10 bg-base-100/50 border-white/30 focus:border-primary/50 text-base-content placeholder:text-base-content/50"
                @keypress="handleKeyPress"
                :disabled="isLoading"
              />
              <button 
                type="button"
                @click="showPassword = !showPassword"
                class="absolute right-3 top-1/2 transform -translate-y-1/2 text-base-content/50 hover:text-base-content transition-colors"
                :disabled="isLoading"
              >
                <Icon :icon="showPassword ? 'material-symbols:visibility-off' : 'material-symbols:visibility'" class="text-lg" />
              </button>
            </div>
          </div>

          <!-- Selezione Realm -->
          <div class="form-control mb-6">
            <label class="label">
              <span class="label-text text-base-content font-medium">Metodo di Autenticazione</span>
            </label>
            <div class="relative">
              <Icon icon="material-symbols:domain" class="absolute left-3 top-1/2 transform -translate-y-1/2 text-base-content/50 z-10" />
              <select 
                v-model="selectedRealm"
                class="select select-bordered w-full pl-10 bg-base-100/50 border-white/30 focus:border-primary/50 text-base-content appearance-none"
                :disabled="isLoading"
              >
                <option v-for="realm in realms" :key="realm.id" :value="realm.name">
                  {{ realm.name }} - {{ realm.description }}
                </option>
              </select>
              <Icon icon="material-symbols:expand-more" class="absolute right-3 top-1/2 transform -translate-y-1/2 text-base-content/50 pointer-events-none" />
            </div>
          </div>

          <!-- Pulsante Login -->
          <button 
            @click="login()"
            class="btn btn-primary w-full gap-2"
            :disabled="isLoading || !username.trim() || !password.trim()"
            :class="{ 'loading': isLoading }"
          >
            <Icon v-if="!isLoading" icon="material-symbols:login" class="text-lg" />
            {{ isLoading ? 'Accesso in corso...' : 'Accedi' }}
          </button>

          <!-- Informazioni aggiuntive -->
          <div class="text-center mt-6 text-sm text-base-content/60">
            
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div 
        class="text-center mt-8 transition-all duration-300 ease-out"
        :class="animationComplete ? 'opacity-100' : 'opacity-0'"
      >
       
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Animazione semplice e veloce per gli elementi */
.logo-container {
  transition: all 0.3s ease-out;
}

/* Animazione per il focus degli input */
.input:focus {
  transform: scale(1.02);
  transition: all 0.2s ease-out;
}

/* Animazione hover per il pulsante */
.btn:not(:disabled):hover {
  transform: translateY(-2px);
  transition: all 0.2s ease-out;
}

/* Effetto glow per il logo */
.logo-container .rounded-2xl {
  box-shadow: 0 10px 25px rgba(99, 102, 241, 0.3);
}

/* Animazione per la card di login */
.liquid-glass-card {
  backdrop-filter: blur(20px);
  box-shadow: 
    0 25px 50px -12px rgba(0, 0, 0, 0.25),
    0 0 20px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}
</style>
