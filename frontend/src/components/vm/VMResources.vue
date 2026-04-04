<script setup lang="ts">
import { onMounted, onBeforeUnmount } from 'vue'
import { ref } from 'vue'
import { api } from '@/lib/api'
import type { VM } from '@/types'
import CreateNew from '@/components/CreateNew.vue'
import { useToastService } from '@/composables/useToast'
import { useUserResources } from '@/composables/userResources'

const { fetchUserResources, userFreeResources } = useUserResources(api)

const { error: toastError } = useToastService()

const $props = defineProps<{
  vm: VM
}>()

const cores = ref($props.vm.cores)
const ram = ref($props.vm.ram)
const disk = ref($props.vm.disk)

const $emit = defineEmits(['update-vm'])

function updateResources() {
  api
    .patch(`/vm/${$props.vm.id}/resources`, {
      cores: cores.value,
      ram: ram.value,
      disk: disk.value,
    })
    .then(() => {
      $emit('update-vm')
    })
    .catch((err) => {
      toastError('Failed to update resources: ' + err.response.data)
      console.error('Failed to update resources:', err)
    })
}

let intervalID: number | null = null

onMounted(() => {
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
    <CreateNew title="Modify Resources" :hideCreate="true" :create="updateResources" :open="true">
      <div class="grid grid-cols-3 gap-4">
        <div>
          <label for="cores">CPU Cores</label>
          <div>
            <div class="join w-full">
              <input
                type="number"
                id="cores"
                v-model="cores"
                class="input join-item validator w-full rounded-l-lg border p-2"
                min="1"
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
                min="1024"
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
                :min="$props.vm.disk"
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
      </div>
    </CreateNew>

    <div class="flex flex-col gap-2">
      <div>CPU Cores: {{ $props.vm.cores }}</div>
      <div>RAM: {{ $props.vm.ram }}</div>
      <div>Disk: {{ $props.vm.disk }}</div>
    </div>
  </div>
</template>
