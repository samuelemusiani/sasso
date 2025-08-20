<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/lib/api'
import { defineEmits } from 'vue'

const emit = defineEmits(['realmAdded'])

const name = ref('')
const description = ref('')
const url = ref('')
const baseDN = ref('')
const bindDN = ref('')
const bindPassword = ref('')

function addRealm() {
  const realmData = {
    name: name.value,
    description: description.value,
    url: url.value,
    baseDN: baseDN.value,
    bindDN: bindDN.value,
    type: 'ldap',
    password: bindPassword.value,
  }

  // Here you would typically send the data to your backend API
  // For example:
  api
    .post('/admin/realms', realmData)
    .then((response) => {
      console.log('Realm created:', response.data)
      emit('realmAdded')
    })
    .catch((error) => {
      console.error('Error creating realm:', error)
    })
}
</script>

<template>
  <div>
    <div class="text-lg font-bold">LDAP Realm</div>
    <form class="max-w-96">
      <label class="block mb-2 text-gray-800">Name</label>
      <input
        v-model="name"
        type="text"
        placeholder="My LDAP Realm"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Description</label>
      <input
        v-model="description"
        type="text"
        placeholder="A description of my LDAP Realm"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">URL</label>
      <input
        v-model="url"
        type="text"
        placeholder="ldap://server.test.com:389"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Base DN</label>
      <input
        v-model="baseDN"
        type="text"
        placeholder="dc=example,dc=com"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Bind DN</label>
      <input
        v-model="bindDN"
        type="text"
        placeholder="cn=admin,dc=example,dc=com"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Bind Password</label>
      <input
        v-model="bindPassword"
        type="password"
        placeholder="Your bind password"
        class="border p-2 rounded w-full mb-2"
      />

      <button
        @click.prevent="addRealm()"
        class="bg-blue-500 hover:bg-blue-400 text-white font-bold py-2 px-4 rounded w-full"
      >
        Create Realm
      </button>
    </form>
  </div>
</template>
