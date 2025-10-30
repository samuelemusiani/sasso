<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { VM } from '@/types'
import { getStatusClass } from '@/const'
import { formatDate, isVMExpired, vmWillExpire } from '@/lib/utils'
import { api } from '@/lib/api'
import { useLoadingStore } from '@/stores/loading'
import { useRouter } from 'vue-router'

const $props = defineProps<{
  vm: VM
}>()

const router = useRouter()
const $emit = defineEmits(['update-vm', 'status-change'])

const possibleExtensions = [1, 2, 3]
const acctuallyPossibleExtensions = computed(() => {
  return vmWillExpire($props.vm.lifetime, possibleExtensions).possible_extend_by
})

const extendBy = ref(1)

watch(
  () => acctuallyPossibleExtensions.value,
  (newVal) => {
    if (!newVal.includes(extendBy.value)) {
      extendBy.value = newVal[0] || 1
    }
  },
  { immediate: true },
)

const loading = useLoadingStore()
const isLoading = (vmId: number, action: string) => loading.is('vm', vmId, action)

function updateLifetime(vmid: number, extend_by: number) {
  api
    .patch(`/vm/${$props.vm.id}/lifetime`, { extend_by })
    .then(() => {
      $emit('update-vm')
    })
    .catch((err) => {
      console.error('Failed to update VM lifetime:', err)
    })
}

function startVM(vmid: number) {
  loading.start('vm', vmid, 'start')
  api
    .post(`/vm/${vmid}/start`)
    .then(() => {
      $emit('status-change', 'running')
    })
    .catch((err) => console.error('Failed to start VM:', err))
    .finally(() => loading.stop('vm', vmid, 'start'))
}

function stopVM(vmid: number) {
  loading.start('vm', vmid, 'stop')
  api
    .post(`/vm/${vmid}/stop`)
    .then(() => {
      $emit('status-change', 'stopped')
    })
    .catch((err) => console.error('Failed to stop VM:', err))
    .finally(() => loading.stop('vm', vmid, 'stop'))
}

function restartVM(vmid: number) {
  loading.start('vm', vmid, 'restart')
  api
    .post(`/vm/${vmid}/restart`)
    .then(() => {
      $emit('status-change', 'running')
    })
    .catch((err) => console.error('Failed to restart VM:', err))
    .finally(() => loading.stop('vm', vmid, 'restart'))
}

function deleteVM(vmid: number) {
  if (confirm(`Are you sure you want to delete VM ${vmid}?`)) {
    api
      .delete(`/vm/${vmid}`)
      .then(() => {
        router.push('/vm')
      })
      .catch((err) => {
        console.error('Failed to delete VM:', err)
      })
  }
}

const disableDelete = computed(() => {
  if ($props.vm.group_role === 'member') return true
  const deleteStates = ['stopped', 'running', 'unknown']

  if (deleteStates.includes($props.vm.status)) return false

  return true
})
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="mb-4 text-2xl font-bold">Name: {{ $props.vm.name }}</div>
    <div class="mb-2 font-semibold text-gray-600">ID: {{ $props.vm.id }}</div>
    <div>Notes: {{ $props.vm.notes }}</div>
    <div class="flex gap-4">
      <div>
        Status:
        <span :class="getStatusClass($props.vm.status)" class="font-semibold capitalize">
          {{ $props.vm.status }}
        </span>
      </div>
      <div class="*:btn-sm col-span-2 grid grid-cols-3 items-center gap-2 xl:col-span-1">
        <button
          v-if="vm.status === 'stopped'"
          @click="startVM(vm.id)"
          :disabled="
            isLoading(vm.id, 'start') || isVMExpired(vm.lifetime) || vm.group_role == 'member'
          "
          class="btn btn-success btn-outline col-span-2 rounded-lg"
        >
          <span v-if="isLoading(vm.id, 'start')" class="loading loading-spinner loading-xs"></span>
          <IconVue v-else icon="material-symbols:play-arrow" class="text-lg" />
          <span class="hidden lg:inline">Start</span>
        </button>

        <button
          v-if="vm.status === 'running'"
          @click="stopVM(vm.id)"
          :disabled="isLoading(vm.id, 'stop') || vm.group_role == 'member'"
          class="btn btn-warning btn-outline rounded-lg"
        >
          <span v-if="isLoading(vm.id, 'stop')" class="loading loading-spinner loading-xs"></span>
          <IconVue v-else icon="material-symbols:stop" class="text-lg" />
          <span class="hidden lg:inline">Stop</span>
        </button>

        <button
          v-if="vm.status === 'running'"
          @click="restartVM(vm.id)"
          :disabled="isLoading(vm.id, 'restart') || vm.group_role == 'member'"
          class="btn btn-info btn-outline rounded-lg"
        >
          <span
            v-if="isLoading(vm.id, 'restart')"
            class="loading loading-spinner loading-xs"
          ></span>
          <IconVue v-else icon="codicon:debug-restart" class="text-lg" />
          <span class="hidden lg:inline">Restart</span>
        </button>
      </div>
    </div>
    <div>
      Lifetime:
      <span>
        {{ formatDate($props.vm.lifetime) }}
      </span>
    </div>

    <div
      v-show="
        vmWillExpire(vm.lifetime, possibleExtensions).will_expire && vm.group_role != 'member'
      "
      class="*:btn-sm col-span-2 grid grid-cols-3 items-center gap-2 xl:col-span-1"
    >
      <select class="select" v-model.number="extendBy">
        <option v-for="option in acctuallyPossibleExtensions" :key="option" :value="option">
          {{ option }} month<span v-if="option > 1">s</span>
        </option>
      </select>
      <button @click="updateLifetime(vm.id, extendBy)" class="btn btn-primary btn-sm rounded-lg">
        <IconVue icon="material-symbols:update" class="text-lg" />
        <span class="hidden md:inline">Extend</span>
      </button>
    </div>

    <div>
      Include Global SSH Keys:
      <span class="font-bold">
        {{ $props.vm.include_global_ssh_keys }}
      </span>
    </div>

    <div class="divider text-error my-4 font-bold">Danger Zone</div>

    <button
      @click="deleteVM(vm.id)"
      :disabled="disableDelete"
      class="btn btn-error btn-outline w-70 rounded-lg"
    >
      <IconVue icon="material-symbols:delete" class="text-lg" />
      <span class="hidden lg:inline">Delete</span>
    </button>
  </div>
</template>
