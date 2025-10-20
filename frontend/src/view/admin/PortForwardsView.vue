<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { AdminPortForward } from '@/types'
import { api } from '@/lib/api'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'

const pfs = ref<AdminPortForward[]>([])

function fetchPortForwards() {
  api
    .get('/admin/port-forwards')
    .then((res) => {
      res.data.sort((a: AdminPortForward, b: AdminPortForward) => a.id - b.id)
      pfs.value = res.data as AdminPortForward[]
    })
    .catch((err) => {
      console.error('Failed to fetch Port Forwards:', err)
    })
}

function approvePortForward(id: number, approve: boolean) {
  api
    .put(`/admin/port-forwards/${id}`, {
      approve: approve,
    })
    .then(() => {
      fetchPortForwards()
    })
    .catch((err) => {
      console.error('Failed to delete Port Forward:', err)
    })
}

onMounted(() => {
  fetchPortForwards()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <AdminBreadcrumbs />
    <div class="overflow-x-auto">
      <table class="table min-w-full divide-y">
        <thead class="">
          <tr>
            <th scope="col" class="">Out Port</th>
            <th scope="col" class="">Destination Port</th>
            <th scope="col" class="">Destination IP</th>
            <th scope="col" class="">Name</th>
            <th scope="col" class="">Is Group</th>
            <th scope="col" class="relative px-6 py-3">
              <span class="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody class="divide-y">
          <tr v-for="pf in pfs" :key="pf.id">
            <td class="">{{ pf.out_port }}</td>
            <td class="">{{ pf.dest_port }}</td>
            <td class="">{{ pf.dest_ip }}</td>
            <td class="">{{ pf.name }}</td>
            <td class="">{{ pf.is_group || false }}</td>
            <td class="flex text-right text-sm font-medium">
              <button
                @click="approvePortForward(pf.id, !pf.approved)"
                class="btn grow rounded-lg p-2"
                :class="pf.approved ? 'btn-error' : 'btn-success'"
              >
                {{ pf.approved ? 'Revoke' : 'Approve' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
