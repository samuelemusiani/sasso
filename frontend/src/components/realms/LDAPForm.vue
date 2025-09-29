<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/lib/api'
import type { LDAPRealm } from '@/types'

const $emit = defineEmits(['realmAdded'])
const $props = defineProps<{
  realm?: LDAPRealm
}>()

const name = ref($props.realm ? $props.realm.name : '')
const description = ref($props.realm ? $props.realm.description : '')
const url = ref($props.realm ? $props.realm.url : '')
const baseDN = ref($props.realm ? $props.realm.base_dn : '')
const bindDN = ref($props.realm ? $props.realm.bind_dn : '')
const bindPassword = ref('')
const filter = ref($props.realm ? $props.realm.filter || '' : '')

const editing = ref(!!$props.realm)

function addRealm() {
  const realmData = {
    name: name.value,
    description: description.value,
    url: url.value,
    baseDN: baseDN.value,
    bindDN: bindDN.value,
    type: 'ldap',
    password: bindPassword.value,
    filter: filter.value,
  }

  api
    .post('/admin/realms', realmData)
    .then((response) => {
      console.log('Realm created:', response.data)
      $emit('realmAdded')
    })
    .catch((error) => {
      console.error('Error creating realm:', error)
    })
}
</script>

<template>
  <div>
    <form class="space-y-4">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Nome del Realm -->
        <div class="form-control">
          <label class="label">
            <span class="label-text font-medium text-base-content">Nome del Realm</span>
          </label>
          <input
            v-model="name"
            type="text"
            placeholder="Il mio Realm LDAP"
            class="input input-bordered w-full"
          />
        </div>

        <!-- URL del Server -->
        <div class="form-control">
          <label class="label">
            <span class="label-text font-medium text-base-content">URL del Server</span>
          </label>
          <input
            v-model="url"
            type="text"
            placeholder="ldap://server.example.com:389"
            class="input input-bordered w-full"
          />
        </div>
      </div>

      <!-- Descrizione -->
      <div class="form-control">
        <label class="label">
          <span class="label-text font-medium text-base-content">Descrizione</span>
        </label>
        <input
          v-model="description"
          type="text"
          placeholder="Descrizione del realm LDAP"
          class="input input-bordered w-full"
        />
      </div>

      <!-- Filtro LDAP -->
      <div class="form-control">
        <label class="label">
          <span class="label-text font-medium text-base-content">Filtro LDAP</span>
          <span class="label-text-alt text-base-content/60"
            >Filtro per selezionare gli utenti (opzionale)</span
          >
        </label>
        <input
          v-model="filter"
          type="text"
          placeholder="(objectClass=person)"
          class="input input-bordered w-full font-mono text-sm"
        />
        <div class="label">
          <span class="label-text-alt text-base-content/50"
            >Esempio: (memberOf=cn=admins,ou=groups,dc=example,dc=com)</span
          >
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Base DN -->
        <div class="form-control">
          <label class="label">
            <span class="label-text font-medium text-base-content">Base DN</span>
            <span class="label-text-alt text-base-content/60">Distinguished Name di base</span>
          </label>
          <input
            v-model="baseDN"
            type="text"
            placeholder="dc=example,dc=com"
            class="input input-bordered w-full font-mono text-sm"
          />
        </div>

        <!-- Bind DN -->
        <div class="form-control">
          <label class="label">
            <span class="label-text font-medium text-base-content">Bind DN</span>
            <span class="label-text-alt text-base-content/60">Account per l'autenticazione</span>
          </label>
          <input
            v-model="bindDN"
            type="text"
            placeholder="cn=admin,dc=example,dc=com"
            class="input input-bordered w-full font-mono text-sm"
          />
        </div>
      </div>

      <!-- Bind Password -->
      <div class="form-control">
        <label class="label">
          <span class="label-text font-medium text-base-content">Password di Bind</span>
          <span class="label-text-alt text-base-content/60" v-if="$props.realm">
            Lascia vuoto per mantenere la password esistente
          </span>
        </label>
        <input
          v-model="bindPassword"
          type="password"
          :placeholder="$props.realm ? 'Password non modificata' : 'Password per l\'autenticazione'"
          class="input input-bordered w-full"
        />
      </div>

      <!-- Pulsante Create solo se non in editing mode (per compatibilità) -->
      <button @click.prevent="addRealm()" v-if="!editing" class="btn btn-primary w-full">
        Crea Realm LDAP
      </button>
    </form>
  </div>
</template>
