<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Realm } from '@/types'
import RealmsMultiplexer from '@/components/realms/RealmsMultiplexer.vue'

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
    <div>Admin realm view for <b>sasso</b>!</div>
    <RouterLink
      class="bg-gray-400 hover:bg-gray-300 p-2 rounded-lg w-64 block text-center"
      to="/admin"
    >
      Back to Admin Panel
    </RouterLink>
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
    <table class="w-full mt-2 p-2" v-show="!addingRealm">
      <thead>
        <tr class="bg-cyan-500">
          <th class="p-2 border-y border-black border-l">ID</th>
          <th class="p-2 border-y border-black">Name</th>
          <th class="p-2 border-y border-black">Description</th>
          <th class="p-2 border-y border-black">Type</th>
          <th class="p-2 border-y border-black border-r"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="realm in realms" :key="realm.id" class="odd:bg-blue-100 even:bg-blue-200">
          <td class="p-2 text-center border-y border-black border-l">{{ realm.id }}</td>
          <td class="p-2 text-center border-y border-black">{{ realm.name }}</td>
          <td class="p-2 text-center border-y border-black">{{ realm.description }}</td>
          <td class="p-2 text-center border-y border-black">{{ realm.type }}</td>
          <td class="p-2 text-center border-y border-black border-r">
            <div class="flex justify-evenly" v-show="realm.type != 'local'">
              <RouterLink class="text-blue-500 hover:underline" :to="`/admin/realms/${realm.id}`"
                >Edit</RouterLink
              >
              <button class="text-red-500 hover:underline" @click="deleteRealm(realm.id)">
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
