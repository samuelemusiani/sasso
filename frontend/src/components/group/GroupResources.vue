<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from '@/lib/api'
import UserStats from '@/components/UserStats.vue'
import CreateNew from '@/components/CreateNew.vue'
import { useToastService } from '@/composables/useToast'
import type { Group, GroupMember } from '@/types'
import { useUserResources } from '@/composables/userResources'

const { fetchUserResources, userFreeResources } = useUserResources(api)

const $props = defineProps<{
  group: Group
  me: GroupMember
}>()

const $emit = defineEmits(['update-group'])

const { success: toastSuccess } = useToastService()

const cores = ref(0)
const ram = ref(0)
const disk = ref(0)
const nets = ref(0)
const error = ref('')

const stats = ref()

// Used to update the resource fields when the component is loaded or when the group/me props change
// So 'update resource' have the current values of the resources instead of 0
watch([$props.group, $props.me], ([newGroup, newMe]) => {
  if (newGroup && newMe) {
    const myResource = newGroup.resources?.find((r) => r.user_id === newMe.user_id)
    if (myResource) {
      cores.value = myResource.cores
      ram.value = myResource.ram
      disk.value = myResource.disk
      nets.value = myResource.nets
    }
  }
})

const addOrUpdateResources = computed(() => {
  return $props.group.resources?.find((r) => r.user_id === $props.me.user_id) !== undefined
})

async function saveResources() {
  return api
    .put(`/groups/${$props.group.id}/resources`, {
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
      nets: nets.value,
    })
    .then(() => {
      toastSuccess('Resources saved successfully.')
      $emit('update-group')
      fetchResourceStats()
      return true
    })
    .catch((err) => {
      console.error('Failed to save resources:', err)
      error.value = `Failed to save resources. ${err.response?.data}`
      return false
    })
}

async function fetchResourceStats() {
  api
    .get(`/groups/${$props.group.id}/resources`)
    .then((res) => {
      const data = res.data
      stats.value = [
        {
          item: 'CPU',
          active: data.active_vms_cores,
          max: data.max_cores,
          allocated: data.allocated_cores,
        },
        {
          item: 'RAM',
          active: data.active_vms_ram / 1024,
          max: data.max_ram / 1024,
          allocated: data.allocated_ram / 1024,
        },
        {
          item: 'Disk',
          active: data.active_vms_disk,
          max: data.max_disk,
          allocated: data.allocated_disk,
        },
        {
          item: 'Net',
          active: -1,
          max: data.max_nets,
          allocated: data.allocated_nets,
        },
      ]
    })
    .catch((err) => {
      console.error('Failed to fetch resource stats:', err)
    })
}

let intervalID: number | null = null

onMounted(() => {
  fetchResourceStats()
  fetchUserResources()
  intervalID = setInterval(() => {
    fetchUserResources()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalID) {
    clearInterval(intervalID)
  }
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <UserStats v-if="stats" :stats="stats" />

    <CreateNew
      :hideCreate="addOrUpdateResources"
      :title="(addOrUpdateResources ? 'Update ' : '') + 'Resources'"
      :create="saveResources"
      :error="error"
      :close-on-create="true"
    >
      <div class="grid grid-cols-4 gap-4">
        <div>
          <label for="cores">CPU Cores</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="cores"
                v-model="cores"
                class="input join-item validator w-full rounded-l-lg border p-2"
                :max="userFreeResources?.free_cpu"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available CPU cores</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_cpu }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div>
          <label for="ram">RAM (MB)</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="ram"
                v-model="ram"
                class="input joint-item validator w-full rounded-l-lg border p-2"
                :max="userFreeResources?.free_ram"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available RAM</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_ram }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div>
          <label for="disk">Disk (GB)</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="disk"
                v-model="disk"
                class="input join-item validator w-full rounded-l-lg border p-2"
                :max="userFreeResources?.free_disk"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip tooltip-top flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available Disk</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_disk }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div>
          <label for="net">Nets</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="net"
                v-model="nets"
                class="input join-item validator w-full rounded-l-lg border p-2"
                :max="userFreeResources?.free_nets"
              />
              <div
                class="join-item bg-base-300 border-base-content/30 rounded-r-lg border p-2 text-sm"
              >
                <div class="tooltip tooltip-top flex items-center">
                  <div class="tooltip-content rounded-lg border p-2">Max available Nets</div>
                  <span class="mr-1 opacity-50">/ </span>
                  {{ userFreeResources?.free_nets }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </CreateNew>
  </div>
</template>
