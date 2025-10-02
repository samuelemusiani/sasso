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
    <div class="text-2xl font-bold">LDAP Realm</div>
    <p>{{ $props.realm ? 'Edit LDAP Realm' : 'Add LDAP Realm' }}</p>
    <form class="flex gap-2 flex-col mt-2">
      <label class="label">Name</label>
      <input
        v-model="name"
        type="text"
        placeholder="My LDAP Realm"
        class="input rounded-lg w-full"
      />

      <label class="label">Description</label>
      <input
        v-model="description"
        type="text"
        placeholder="A description of my LDAP Realm"
        class="input rounded-lg w-full"
      />

      <label class="label">URL</label>
      <input
        v-model="url"
        type="text"
        placeholder="ldap://server.test.com:389"
        class="input rounded-lg w-full"
      />

      <label class="label">User Base DN</label>
      <input
        v-model="userBaseDN"
        type="text"
        placeholder="ou=people,dc=example,dc=com"
        class="input rounded-lg w-full"
      />

      <label class="label">Group Base DN</label>
      <input
        v-model="groupBaseDN"
        type="text"
        placeholder="ou=group,dc=example,dc=com"
        class="input rounded-lg w-full"
      />

      <label class="label">Admin Group</label>
      <input
        v-model="adminGroup"
        type="text"
        placeholder="sasso_admin"
        class="input rounded-lg w-full"
      />

      <label class="label">Maintainer Group</label>
      <input
        v-model="maintainerGroup"
        type="text"
        placeholder="sasso_maintainer"
        class="input rounded-lg w-full"
      />

      <label class="label">Bind DN</label>
      <input
        v-model="bindDN"
        type="text"
        placeholder="cn=admin,dc=example,dc=com"
        class="input rounded-lg w-full"
      />

      <label class="label">Bind Password</label>
      <input
        v-model="bindPassword"
        type="password"
        :placeholder="$props.realm ? 'Unchanged' : 'Pour bind password'"
        class="input rounded-lg w-full"
      />
      <button @click.prevent="addRealm()" v-if="!editing" class="btn btn-primary w-full rounded-lg">
        Create Realm
      </button>
      <div v-else class="flex gap-2 grow">
        <RouterLink
          to="/admin/realms"
          class="btn btn-error btn-outline text-center w-1/2 rounded-lg"
        >
          Cancel
        </RouterLink>
        <button @click.prevent="updateRealm()" class="btn btn-primary text-center w-1/2 rounded-lg">
          Update Realm
        </button>
      </div>
    </form>
  </div>
</template>
