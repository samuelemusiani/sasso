<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Realm } from '@/types'
import RealmsMultiplexer from '@/components/realms/RealmsMultiplexer.vue'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'

const realms = ref<Realm[]>([])

const addingRealm = ref(false)
const addingType = ref('ldap')

function fetchRealms() {
  api
    .get('/admin/realms')
    .then((res) => {
      realms.value = res.data as Realm[]
    })
    .catch((err) => {
      console.error('Failed to fetch realms:', err)
    })
}

function realmAdded() {
  addingRealm.value = false
  fetchRealms()
}

function deleteRealm(id: number) {
  if (!confirm('Are you sure you want to delete this realm?')) {
    return
  }
  api
    .delete(`/admin/realms/${id}`)
    .then(() => {
      console.log(`Realm ${id} deleted successfully`)
      fetchRealms()
    })
    .catch((err) => {
      console.error(`Failed to delete realm ${id}:`, err)
    })
}

onMounted(() => {
  fetchRealms()
})
</script>

<template>
  <div class="p-2">
    <AdminBreadcrumbs />
    <button
      class="bg-blue-400 hover:bg-blue-300 p-2 rounded-lg w-64 block text-center"
      @click="addingRealm = true"
      v-show="!addingRealm"
    >
      Add LDAP Realm
    </button>
    <button
      class="bg-red-400 hover:bg-red-300 p-2 rounded-lg w-64 block text-center"
      @click="addingRealm = false"
      v-show="addingRealm"
    >
      Cancel
    </button>
    <table class="table w-full mt-2 p-2" v-show="!addingRealm">
      <thead>
        <tr class="">
          <th class="">Name</th>
          <th class="">Description</th>
          <th class="">Type</th>
          <th class=""></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="realm in realms" :key="realm.id" class="odd:bg-base-100 even:bg-base-200">
          <td class="">{{ realm.name }}</td>
          <td class="">{{ realm.description }}</td>
          <td class="">{{ realm.type }}</td>
          <td class="">
            <div class="flex justify-start gap-2" v-show="realm.type != 'local'">
              <RouterLink class="btn btn-primary" :to="`/admin/realms/${realm.id}`"
                >Edit</RouterLink
              >
              <button class="btn btn-error btn-outline" @click="deleteRealm(realm.id)">
                Delete
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>

    <RealmsMultiplexer
      class="mt-4"
      v-show="addingRealm"
      :adding="addingRealm"
      :type="addingType"
      @realm-added="realmAdded"
    />

    <router-view class="mt-4" />
  </div>
</template>
