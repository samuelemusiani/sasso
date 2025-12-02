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
const loginFilter = ref($props.realm ? $props.realm.login_filter : '')
const maintainerFilter = ref($props.realm ? $props.realm.maintainer_filter : '')
const adminFilter = ref($props.realm ? $props.realm.admin_filter : '')
const mailAttribute = ref($props.realm ? $props.realm.mail_attribute : '')

const editing = ref(!!$props.realm)

function formLDAPRealm(): LDAPRealm {
  const realmData: LDAPRealm = {
    id: 0,
    name: name.value,
    description: description.value,
    url: url.value,
    user_base_dn: userBaseDN.value,
    group_base_dn: groupBaseDN.value,
    bind_dn: bindDN.value,
    type: 'ldap',
    password: bindPassword.value,
    login_filter: loginFilter.value,
    maintainer_filter: maintainerFilter.value,
    admin_filter: adminFilter.value,
    mail_attribute: mailAttribute.value,
  }
  return realmData
}

function addRealm() {
  const realmData = formLDAPRealm()

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
  const realmData = formLDAPRealm()
  if (!bindPassword.value) {
    delete realmData.password
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
    <form class="mt-2 flex flex-col gap-2">
      <label class="label">Name</label>
      <input
        v-model="name"
        type="text"
        placeholder="My LDAP Realm"
        class="input w-full rounded-lg"
      />

      <label class="label">Description</label>
      <input
        v-model="description"
        type="text"
        placeholder="A description of my LDAP Realm"
        class="input w-full rounded-lg"
      />

      <label class="label">URL</label>
      <input
        v-model="url"
        type="text"
        placeholder="ldap://server.test.com:389"
        class="input w-full rounded-lg"
      />

      <label class="label">User Base DN</label>
      <input
        v-model="userBaseDN"
        type="text"
        placeholder="ou=people,dc=example,dc=com"
        class="input w-full rounded-lg"
      />

      <label class="label">Email attribute</label>
      <input
        v-model="mailAttribute"
        type="text"
        placeholder="mail"
        class="input w-full rounded-lg"
      />

      <label class="label">Login Filter</label>
      <input
        v-model="loginFilter"
        type="text"
        placeholder="(&(objectClass=person)(uid={{username}}))"
        class="input w-full rounded-lg"
      />

      <label class="label">Group Base DN</label>
      <input
        v-model="groupBaseDN"
        type="text"
        placeholder="ou=group,dc=example,dc=com"
        class="input w-full rounded-lg"
      />

      <label class="label">Maintainer Filter</label>
      <input
        v-model="maintainerFilter"
        type="text"
        placeholder="(&(objectClass=groupOfNames)(cn=sasso_maintainer)(member={{user_dn}}))"
        class="input w-full rounded-lg"
      />
      <label class="label">Admin Filter</label>
      <input
        v-model="adminFilter"
        type="text"
        placeholder="(&(objectClass=groupOfNames)(cn=sasso_admin)(member={{user_dn}}))"
        class="input w-full rounded-lg"
      />

      <label class="label">Bind DN</label>
      <input
        v-model="bindDN"
        type="text"
        placeholder="cn=admin,dc=example,dc=com"
        class="input w-full rounded-lg"
      />

      <label class="label">Bind Password</label>
      <input
        v-model="bindPassword"
        type="password"
        :placeholder="$props.realm ? 'Unchanged' : 'Pour bind password'"
        class="input w-full rounded-lg"
      />
      <button @click.prevent="addRealm()" v-if="!editing" class="btn btn-primary w-full rounded-lg">
        Create Realm
      </button>
      <div v-else class="flex grow gap-2">
        <RouterLink
          to="/admin/realms"
          class="btn btn-error btn-outline w-1/2 rounded-lg text-center"
        >
          Cancel
        </RouterLink>
        <button @click.prevent="updateRealm()" class="btn btn-primary w-1/2 rounded-lg text-center">
          Update Realm
        </button>
      </div>
    </form>
  </div>
</template>
