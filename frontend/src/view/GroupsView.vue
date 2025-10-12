<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Group } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'

const groups = ref<Group[]>([])
const name = ref('')
const description = ref('')

function fetchGroups() {
  api
    .get('/groups')
    .then((res) => {
      const tmp = res.data.sort((a: Group, b: Group) => a.id - b.id)
      groups.value = tmp as Group[]
    })
    .catch((err) => {
      console.error('Failed to fetch Groups:', err)
    })
}

function createGroup() {
  api
    .post('/groups', {
      name: name.value,
      description: description.value,
    })
    .then(() => {
      fetchGroups()
      name.value = ''
      description.value = ''
    })
    .catch((err) => {
      console.error('Failed to add Group:', err)
    })
}

function deleteGroup(id: number) {
  if (confirm('Are you sure you want to delete this Group?')) {
    api
      .delete(`/groups/${id}`)
      .then(() => {
        fetchGroups()
      })
      .catch((err) => {
        console.error('Failed to delete Group:', err)
      })
  }
}

onMounted(() => {
  fetchGroups()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <CreateNew title="Group" :create="createGroup">
      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-2">
          <label for="name">Name</label>
          <input type="text" id="name" v-model="name" class="input w-48 rounded-lg border p-2" />
        </div>
        <div>
          <label for="description" class="mb-1 block">Description</label>
          <textarea id="description" v-model="description" class="textarea w-full"></textarea>
        </div>
      </div>
    </CreateNew>

    <table class="table w-full table-auto">
      <thead>
        <tr>
          <th scope="col">Name</th>
          <th scope="col">Description</th>
          <th scope="col" class="">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="g in groups" :key="g.id">
          <td class="whitespace-nowrap">{{ g.name }}</td>
          <td class="whitespace-nowrap">{{ g.description }}</td>
          <td class="flex gap-2">
            <RouterLink
              :to="`/group/${g.id}`"
              class="btn btn-primary btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:edit" class="text-lg" />
              <p class="hidden md:inline">Edit</p>
            </RouterLink>
            <button
              @click="deleteGroup(g.id)"
              class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:delete" class="text-lg" />
              <p class="hidden md:inline">Delete</p>
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
