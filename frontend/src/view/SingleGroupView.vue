<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { Group } from '@/types'
import { api } from '@/lib/api'

const group = ref<Group | null>(null)

const route = useRoute()
const groupId = Number(route.params.id)

function fetchGroup() {
  api
    .get(`/groups/${groupId}`)
    .then((res) => {
      group.value = res.data as Group
    })
    .catch((err) => {
      console.error('Failed to fetch Group:', err)
    })
}

onMounted(() => {
  fetchGroup()
})
</script>

<template>
  <div class="">
    <div class="mb-4 flex items-center justify-between">
      <h1 class="text-2xl font-bold">Group Details</h1>
    </div>

    <div v-if="group" class="rounded-lg border p-4 shadow">
      <h2 class="mb-2 text-xl font-semibold">{{ group.name }}</h2>
      <p class="mb-4 text-gray-600">{{ group.description }}</p>
    </div>
  </div>
</template>
