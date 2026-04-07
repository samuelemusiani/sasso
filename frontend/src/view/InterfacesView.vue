<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { InterfaceExtended } from '@/types'
import { api } from '@/lib/api'
import { getStatusClass } from '@/const'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const interfaces = ref<InterfaceExtended[]>([])

function fetchInterfaces() {
  loading.start('interfaces', null, 'fetch')
  api
    .get('/interfaces')
    .then((res) => {
      res.data.sort((a: InterfaceExtended, b: InterfaceExtended) => a.id - b.id)
      interfaces.value = res.data as InterfaceExtended[]
    })
    .catch((err) => {
      console.error('Failed to fetch interfaces:', err)
      toastError('Failed to fetch interfaces: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('interfaces', null, 'fetch')
    })
}

onMounted(() => {
  fetchInterfaces()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" icon="ph:path"></IconVue>Interfaces
      </h1>
      <HelpButton />
    </div>

    <div v-if="loading.is('interfaces', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="table w-full table-auto">
      <thead>
        <tr>
          <th class="">ID</th>
          <th class="">IP</th>
          <th class="">Gateway</th>
          <th class="">Status</th>
          <th class="">VLAN Tag</th>
          <th class="">Net name</th>
          <th class="">VM name</th>
          <th class="">Owner</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="iface in interfaces"
          :key="iface.id"
          class="hover"
          :class="iface.group_name ? 'bg-base-200' : ''"
        >
          <td class="">{{ iface.id }}</td>
          <td class="">{{ iface.ip_add }}</td>
          <td class="">{{ iface.gateway }}</td>
          <td class="font-semibold capitalize" :class="getStatusClass(iface.status)">
            {{ iface.status }}
          </td>
          <td class="">{{ iface.vlan_tag }}</td>
          <td class="">
            {{ iface.vnet_name }}
          </td>
          <td class="">
            <RouterLink :to="`/vm/${iface.vm_id}/interfaces`" class="hover:underline">
              {{ iface.vm_name }}
              <IconVue class="ml-1 inline-block" icon="nimbus:external-link" />
            </RouterLink>
          </td>
          <td class="">
            <component
              :is="!iface.group_name ? 'span' : 'router-link'"
              :to="`/group/${iface.group_id}`"
              :class="{ 'hover:underline': iface.group_name }"
            >
              {{ iface.group_name ? iface.group_name : 'Me' }}
              <IconVue
                v-if="iface.group_name"
                class="ml-1 inline-block"
                icon="nimbus:external-link"
              />
            </component>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
