<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Group, GroupInvite } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'
import { getPageIcon } from '@/const'
import NotesModal from '@/components/NotesModal.vue'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const groups = ref<Group[]>([])
const name = ref('')
const description = ref('')
const error = ref('')

const invitations = ref<GroupInvite[]>([])

function fetchGroups() {
  loading.start('groups', null, 'fetch')
  api
    .get('/groups')
    .then((res) => {
      const tmp = res.data.sort((a: Group, b: Group) => a.id - b.id)
      groups.value = tmp as Group[]
    })
    .catch((err) => {
      console.error('Failed to fetch Groups:', err)
      toastError('Failed to fetch Groups: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('groups', null, 'fetch')
    })
}

function fetchInvitations() {
  loading.start('groupInvitations', null, 'fetch')
  api
    .get('/groups/invites')
    .then((res) => {
      const tmp = res.data.sort((a: GroupInvite, b: GroupInvite) => a.id - b.id)
      invitations.value = tmp as GroupInvite[]
    })
    .catch((err) => {
      console.error('Failed to fetch Invitations:', err)
      toastError('Failed to fetch Invitations: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('groupInvitations', null, 'fetch')
    })
}

async function createGroup() {
  return api
    .post('/groups', {
      name: name.value,
      description: description.value,
    })
    .then(() => {
      fetchGroups()
      name.value = ''
      description.value = ''
      return true
    })
    .catch((err) => {
      console.error('Failed to add Group:', err)
      error.value = 'Failed to add Group: ' + err.response.data
      return false
    })
}

function manageInvitation(id: number, action: string) {
  api
    .patch(`/groups/invites/${id}`, { action })
    .then(() => {
      fetchInvitations()
      fetchGroups()
    })
    .catch((err) => {
      console.error(`Failed to ${action} invitation:`, err)
    })
}

onMounted(() => {
  fetchGroups()
  fetchInvitations()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" :icon="getPageIcon('groups')"></IconVue>Groups
      </h1>
      <HelpButton />
    </div>
    <CreateNew title="Group" :create="createGroup" :error="error" :close-on-create="true">
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

    <div v-if="loading.is('groups', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="table w-full">
      <thead>
        <tr>
          <th scope="col">Name</th>
          <th scope="col">Description</th>
          <th scope="col" class=""></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="g in groups" :key="g.id">
          <td class="min-w-40 text-lg font-semibold">{{ g.name }}</td>
          <td>
            <div>
              <NotesModal
                :title="`Description for &quot;${g.name ?? ''}&quot;`"
                :body="g.description"
              />
            </div>
          </td>
          <td class="flex justify-end gap-2">
            <RouterLink
              :to="`/group/${g.id}`"
              class="btn btn-primary btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:edit" class="text-lg" />
              <p class="hidden md:inline">Manage</p>
            </RouterLink>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="divider my-4"></div>

    <div>
      <h2 class="mb-2 text-xl font-semibold">Group Invitations</h2>
    </div>

    <div v-if="loading.is('groupInvitations', null, 'fetch')" class="grid h-24">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="table w-full table-auto">
      <thead>
        <tr>
          <th scope="col">Name</th>
          <th scope="col">Description</th>
          <th scope="col">Role</th>
          <th scope="col">State</th>
          <th scope="col">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="i in invitations" :key="i.id">
          <td class="whitespace-nowrap">{{ i.group_name }}</td>
          <td class="whitespace-nowrap">{{ i.group_description }}</td>
          <td class="whitespace-nowrap">{{ i.role }}</td>
          <td class="whitespace-nowrap">{{ i.state }}</td>
          <td class="flex gap-2">
            <button
              @click="manageInvitation(i.id, 'accept')"
              class="btn btn-primary btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:edit" class="text-lg" />
              <p class="hidden md:inline">Accept</p>
            </button>
            <button
              @click="manageInvitation(i.id, 'decline')"
              class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:delete" class="text-lg" />
              <p class="hidden md:inline">Decline</p>
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
