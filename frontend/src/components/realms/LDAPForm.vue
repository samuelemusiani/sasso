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
const userBaseDN = ref($props.realm ? $props.realm.user_base_dn : '')
const groupBaseDN = ref($props.realm ? $props.realm.group_base_dn : '')
const bindDN = ref($props.realm ? $props.realm.bind_dn : '')
const bindPassword = ref('')
const adminGroup = ref($props.realm ? $props.realm.admin_group : '')
const maintainerGroup = ref($props.realm ? $props.realm.maintainer_group : '')

const editing = ref(!!$props.realm)

function addRealm() {
  const realmData = {
    name: name.value,
    description: description.value,
    url: url.value,
    user_base_dn: userBaseDN.value,
    group_base_dn: groupBaseDN.value,
    bind_dn: bindDN.value,
    type: 'ldap',
    password: bindPassword.value,
    admin_group: adminGroup.value,
    maintainer_group: maintainerGroup.value,
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

function updateRealm() {
  const realmData = {
    name: name.value,
    description: description.value,
    url: url.value,
    user_base_dn: userBaseDN.value,
    group_base_dn: groupBaseDN.value,
    bind_dn: bindDN.value,
    type: 'ldap',
    password: bindPassword.value,
    admin_group: adminGroup.value,
    maintainer_group: maintainerGroup.value,
  }

  api
    .put(`/admin/realms/${$props.realm?.id}`, realmData)
    .then((response) => {
      console.log('Realm updated:', response.data)
      $emit('realmAdded')
    })
    .catch((error) => {
      console.error('Error updating realm:', error)
    })
}
</script>

<template>
  <div>
    <div class="text-lg font-bold">LDAP Realm</div>
    {{ $props.realm ? 'Edit LDAP Realm' : 'Add LDAP Realm' }}
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

      <label class="block mb-2 text-gray-800">User Base DN</label>
      <input
        v-model="userBaseDN"
        type="text"
        placeholder="ou=people,dc=example,dc=com"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Group Base DN</label>
      <input
        v-model="groupBaseDN"
        type="text"
        placeholder="ou=group,dc=example,dc=com"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Admin Group</label>
      <input
        v-model="adminGroup"
        type="text"
        placeholder="sasso_admin"
        class="border p-2 rounded w-full mb-2"
      />

      <label class="block mb-2 text-gray-800">Mantainer Group</label>
      <input
        v-model="maintainerGroup"
        type="text"
        placeholder="sasso_maintainer"
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
        :placeholder="$props.realm ? 'Unchanged' : 'Pour bind password'"
        class="border p-2 rounded w-full mb-2"
      />

      <button
        @click.prevent="addRealm()"
        v-if="!editing"
        class="bg-blue-500 hover:bg-blue-400 text-white font-bold py-2 px-4 rounded w-full"
      >
        Create Realm
      </button>
      <div v-else class="flex gap-2">
        <RouterLink
          to="/admin/realms"
          class="bg-red-500 hover:bg-red-400 text-white font-bold py-2 px-4 rounded w-full text-center"
        >
          Cancel
        </RouterLink>
        <button
          @click.prevent="updateRealm()"
          class="bg-blue-500 hover:bg-blue-400 text-white font-bold py-2 px-4 rounded w-full text-center"
        >
          Update Realm
        </button>
      </div>
    </form>
  </div>
</template>
