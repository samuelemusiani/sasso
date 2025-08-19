<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Realm } from '@/types'

const realms = ref<Realm[]>([])

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
    <table class="w-full mt-2 p-2">
      <thead>
        <tr class="bg-cyan-500">
          <th class="p-2 border-y border-black border-l">ID</th>
          <th class="p-2 border-y border-black">Name</th>
          <th class="p-2 border-y border-black">Description</th>
          <th class="p-2 border-y border-black">Type</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="realm in realms" :key="realm.id" class="odd:bg-blue-100 even:bg-blue-200">
          <td class="p-2 text-center border-y border-black border-l">{{ realm.id }}</td>
          <td class="p-2 text-center border-y border-black">{{ realm.name }}</td>
          <td class="p-2 text-center border-y border-black">{{ realm.description }}</td>
          <td class="p-2 text-center border-y border-black">{{ realm.type }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
