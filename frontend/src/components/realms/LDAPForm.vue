<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/lib/api'
import type { LDAPRealm } from '@/types'
import { useToastService } from '@/composables/useToast'

const $emit = defineEmits(['realmAdded'])
const $props = defineProps<{
  realm?: LDAPRealm
}>()

const { error: toastError, success: toastSuccess } = useToastService()

const name = ref($props.realm ? $props.realm.name : '')
const description = ref($props.realm ? $props.realm.description : '')
const url = ref($props.realm ? $props.realm.url : '')
const userBaseDN = ref($props.realm ? $props.realm.user_base_dn : '')
const bindDN = ref($props.realm ? $props.realm.bind_dn : '')
const bindPassword = ref('')
const loginFilter = ref($props.realm ? $props.realm.login_filter : '')
const maintainerGroupDN = ref($props.realm ? $props.realm.maintainer_group_dn : '')
const adminGroupDN = ref($props.realm ? $props.realm.admin_group_dn : '')
const mailAttribute = ref($props.realm ? $props.realm.mail_attribute : '')

const editing = ref(!!$props.realm)

function formLDAPRealm(): LDAPRealm {
  const realmData: LDAPRealm = {
    id: 0,
    name: name.value,
    description: description.value,
    url: url.value,
    user_base_dn: userBaseDN.value,
    bind_dn: bindDN.value,
    type: 'ldap',
    password: bindPassword.value,
    login_filter: loginFilter.value,
    maintainer_group_dn: maintainerGroupDN.value,
    admin_group_dn: adminGroupDN.value,
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
      toastSuccess('Realm created successfully')
      $emit('realmAdded')
    })
    .catch((error) => {
      toastError('Error creating realm: ' + error.response.data)
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
      toastSuccess('Realm updated successfully')
      $emit('realmAdded')
    })
    .catch((error) => {
      toastError('Error updating realm: ' + error.response.data)
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

      <label class="label">Maintainer Group DN</label>
      <input
        v-model="maintainerGroupDN"
        type="text"
        placeholder="cn=sasso_maintainer,ou=groups,dc=example,dc=com"
        class="input w-full rounded-lg"
      />

      <label class="label">Admin Group DN</label>
      <input
        v-model="adminGroupDN"
        type="text"
        placeholder="cn=sasso_admin,ou=groups,dc=example,dc=com"
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
