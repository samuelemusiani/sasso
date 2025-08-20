<script setup lang="ts">
import LDAPForm from '@/components/realms/LDAPForm.vue'
import { ref, onMounted } from 'vue'
import { api } from '@/lib/api'
import { useRoute } from 'vue-router'
import type { Realm, LDAPRealm } from '@/types'

const $route = useRoute()
const $emit = defineEmits(['realmAdded'])
const $props = defineProps<{
  adding?: boolean
  type?: string
}>()

const realm = ref<Realm | null>(null)

function realmAdded() {
  $emit('realmAdded')
}

function getRealm() {
  if (!$route.params.id) {
    console.info('No realm ID provided in route params')
    return
  }

  api
    .get(`/admin/realms/${$route.params.id}`)
    .then((res) => {
      realm.value = res.data as Realm
    })
    .catch((err) => {
      console.error('Failed to fetch realm:', err)
    })
}

onMounted(() => {
  if (!$props.adding) {
    getRealm()
  }
  console.log('Multiplexer mounted')
})
</script>

<template>
  <div class="p-4">
    <LDAPForm
      class="w-1/3"
      v-if="(realm && realm.type == 'ldap') || $props.type == 'ldap'"
      @realm-added="realmAdded"
      :realm="realm as LDAPRealm"
    />
  </div>
</template>
