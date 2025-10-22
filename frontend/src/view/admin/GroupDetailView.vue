<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/lib/api'
import type { Group, GroupResource } from '@/types'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'

const route = useRoute()

const groupID = route.params.id

const group = ref<Group | null>(null)
const maxCores = ref<number>(0)
const maxRAM = ref<number>(0)
const maxDisk = ref<number>(0)
const maxNets = ref<number>(0)

function fetchGroup() {
  api
    .get(`/admin/groups/${groupID}`)
    .then((res) => {
      console.log(res.data)
      group.value = res.data as Group
      if (group.value) {
        const admin = group.value.resources?.find((r) => r.username === 'admin')
        maxCores.value = admin?.cores || 0
        maxRAM.value = admin?.ram || 0
        maxDisk.value = admin?.disk || 0
        maxNets.value = admin?.nets || 0
      }
    })
    .catch((err) => {
      console.error('Failed to fetch group:', err)
    })
}

function getResourcesForUser(userId: number): GroupResource | undefined {
  return group.value?.resources?.find((r) => r.user_id === userId)
}

function updateLimits() {
  api
    .put(`/admin/groups/${groupID}/resources`, {
      cores: maxCores.value,
      ram: maxRAM.value,
      disk: maxDisk.value,
      nets: maxNets.value,
    })
    .then(() => {
      alert('User limits updated successfully!')
      fetchGroup()
    })
    .catch((err) => {
      console.error('Failed to update user limits:', err)
      alert('Failed to update user limits.')
    })
}

onMounted(() => {
  fetchGroup()
})
</script>

<template>
  <div class="p-2">
    <AdminBreadcrumbs />
    <h2 class="text-2xl font-bold">Group Details</h2>

    <div v-if="group" class="mt-4">
      <p><strong>ID</strong> {{ group.id }}</p>
      <p><strong>Name</strong> {{ group.name }}</p>
      <p><strong>Description</strong> {{ group.description }}</p>

      <div class="divider my-4"></div>

      <div>
        <h2 class="mb-2 text-xl font-semibold">Group Members</h2>
        <div>
          <template v-if="group?.members?.length === 0">
            <p>No members in this group.</p>
          </template>
          <div v-else class="overflow-x-auto">
            <table class="table w-full">
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Role</th>
                  <th>Cores</th>
                  <th>RAM (MB)</th>
                  <th>Disk (GB)</th>
                  <th>Nets</th>
                </tr>
              </thead>
              <tbody v-if="group?.members">
                <tr
                  v-for="member in group.members"
                  :key="member.user_id"
                  class="odd:bg-base-100 even:bg-base-200"
                >
                  <td>{{ member.username }}</td>
                  <td>{{ member.role }}</td>
                  <td>{{ getResourcesForUser(member.user_id)?.cores || 0 }}</td>
                  <td>{{ getResourcesForUser(member.user_id)?.ram || 0 }}</td>
                  <td>{{ getResourcesForUser(member.user_id)?.disk || 0 }}</td>
                  <td>{{ getResourcesForUser(member.user_id)?.nets || 0 }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <h3 class="mt-6 text-xl font-bold">Resource Limits</h3>
      <form @submit.prevent="updateLimits" class="mt-4 space-y-4">
        <div>
          <label for="maxCores" class="block text-sm font-medium">Max Cores</label>
          <input type="number" id="maxCores" v-model.number="maxCores" class="input rounded-lg" />
        </div>
        <div>
          <label for="maxRAM" class="block text-sm font-medium">Max RAM (MB)</label>
          <input type="number" id="maxRAM" v-model.number="maxRAM" class="input rounded-lg" />
        </div>
        <div>
          <label for="maxDisk" class="block text-sm font-medium">Max Disk (GB)</label>
          <input type="number" id="maxDisk" v-model.number="maxDisk" class="input rounded-lg" />
        </div>
        <div>
          <label for="maxNets" class="block text-sm font-medium">Max Nets</label>
          <input type="number" id="maxNets" v-model.number="maxNets" class="input rounded-lg" />
        </div>
        <button type="submit" class="btn btn-primary rounded-lg">Update Limits</button>
      </form>
    </div>
    <div v-else>
      <p>Loading group details...</p>
    </div>
  </div>
</template>
