<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, computed } from 'vue'
import type { Group, Net } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { getStatusClass } from '@/const'
import { useToastService } from '@/composables/useToast'
import { useLoadingStore } from '@/stores/loading'
import ModalAlert from '@/components/ModalAlert.vue'

const { error: toastError } = useToastService()

const loading = useLoadingStore()

const nets = ref<Net[]>([])
const formNetName = ref('')
const formNetVlanAware = ref(false)
const formNetGroupId = ref<number>()
const error = ref('')
const groups = ref<Group[]>([])

const modifying = ref(false)
const modifyingNetId = ref<number | null>(null)

function fetchNets() {
  api
    .get('/net')
    .then((res) => {
      // nets.value = res.data as Net[]
      res.data.sort((a: Net, b: Net) => a.id - b.id)
      nets.value = res.data as Net[]
    })
    .catch((err) => {
      error.value = 'Failed to fetch nets: ' + err.response.data
      console.error('Failed to fetch nets:', err)
    })
}

function fetchGroups() {
  api.get('/groups').then((res) => {
    groups.value = res.data as Group[]
  })
}

let intervalId: number | null = null

onMounted(() => {
  fetchNets()
  fetchGroups()
  intervalId = setInterval(() => {
    fetchNets()
  }, 5000)
})

onBeforeUnmount(() => {
  if (intervalId) {
    clearInterval(intervalId)
  }
})

interface NetCreationBody {
  name: string
  vlanaware: boolean
  group_id?: number
}

function createOrModifyNet() {
  if (modifying.value) {
    return modifyNet()
  }
  return createNet()
}

function createNet() {
  if (!formNetName.value) {
    error.value = 'Please provide a valid network name'
    return false
  }

  const body: NetCreationBody = {
    name: formNetName.value,
    vlanaware: formNetVlanAware.value,
  }

  if (formNetGroupId.value) {
    body.group_id = formNetGroupId.value
  }

  return api
    .post('/net', body)
    .then(() => {
      formNetName.value = ''
      formNetVlanAware.value = false
      fetchNets()
      return true
    })
    .catch((err) => {
      console.error('Failed to create net:', err)
      error.value = 'Failed to create net: ' + err.response.data
      return false
    })
}

function modifyNet() {
  if (!formNetName.value) {
    error.value = 'Please provide a valid network name'
    return false
  }

  const body: NetCreationBody = {
    name: formNetName.value,
    vlanaware: formNetVlanAware.value,
  }

  return api
    .put(`/net/${modifyingNetId.value}`, body)
    .then(() => {
      toggleModify(-1)
      fetchNets()
      return true
    })
    .catch((err) => {
      console.error('Failed to create net:', err)
      error.value = 'Failed to create net: ' + err.response.data
      return false
    })
}

const showDeleteModal = ref(false)
const netToDelete = ref<number | null>(null)

function preDeleteNet(id: number) {
  netToDelete.value = id
  showDeleteModal.value = true
  loading.start('net', id, 'delete')
}

function deleteNet(id: number) {
  api
    .delete(`/net/${id}`)
    .then(() => {
      console.log(`Network ${id} deleted successfully`)
      fetchNets()
    })
    .catch((err) => {
      toastError(`Failed to delete network: ` + err.response.data)
      console.error(`Failed to delete network ${id}:`, err)
    })
    .finally(() => {
      netToDelete.value = null
      showDeleteModal.value = false
      loading.stop('net', id, 'delete')
    })
}

function cancelDeleteNet(id: number) {
  netToDelete.value = null
  showDeleteModal.value = false
  loading.stop('net', id, 'delete')
}

function toggleModify(id: number) {
  if (id === -1) {
    modifying.value = false
    modifyingNetId.value = null
    formNetName.value = ''
    formNetVlanAware.value = false
    formNetGroupId.value = undefined
    return
  } else {
    modifying.value = true
    modifyingNetId.value = id
    const net = nets.value.find((n) => n.id === id)
    if (net) {
      formNetName.value = net.name
      formNetVlanAware.value = net.vlanaware
      formNetGroupId.value = net.group_id
    }
  }
}

const nonMemberGroups = computed(() => {
  return groups.value.filter((group) => group.role !== 'member')
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" icon="ph:network"></IconVue>Networks
      </h1>
      <HelpButton />
    </div>

    <CreateNew
      :title="modifying ? 'Modify Network' : 'Network'"
      :hideCreate="modifying"
      :create="createOrModifyNet"
      :error="error"
      :open="modifying"
      :close-on-create="true"
      @close="toggleModify(-1)"
    >
      <div class="flex flex-col gap-2">
        <label for="name">Network Name</label>
        <input
          required
          v-model="formNetName"
          type="text"
          placeholder="Network Name"
          class="border-primary rounded-lg border p-2"
        />

        <template v-if="!modifying">
          <label for="group">Group (Optional)</label>
          <select v-model="formNetGroupId" class="select select-bordered">
            <option :value="undefined">Me</option>
            <option v-for="group in nonMemberGroups" :key="group.id" :value="group.id">
              {{ group.name }}
            </option>
          </select>
        </template>

        <label class="flex cursor-pointer items-center gap-3">
          <input v-model="formNetVlanAware" type="checkbox" class="checkbox checkbox-primary" />
          <span class="label-text text-base-content">Enable VLAN support</span>
        </label>
      </div>
    </CreateNew>

    <table class="table w-full table-auto">
      <thead>
        <tr>
          <th>Name</th>
          <th>Owner</th>
          <th>Status</th>
          <th>VlanAware</th>
          <th>Subnet</th>
          <th>Gateway</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="net in nets"
          :key="net.id"
          class="hover"
          :class="net.group_name ? 'bg-base-200' : ''"
        >
          <td>
            <div class="min-w-28 text-lg font-semibold">
              {{ net.name }}
            </div>
          </td>
          <td>{{ net.group_name ? net.group_name : 'Me' }}</td>
          <td>
            <div class="font-semibold capitalize" :class="getStatusClass(net.status)">
              {{ net.status }}
            </div>
          </td>
          <td>{{ net.vlanaware }}</td>
          <td>
            <div>
              <template v-if="net.subnet">{{ net.subnet }}</template>
              <div v-else class="flex items-center">
                <span class="loading loading-dots loading-sm"></span>
              </div>
            </div>
          </td>
          <td>
            <div>
              <template v-if="net.gateway">{{ net.gateway }}</template>
              <div v-else class="flex items-center">
                <span class="loading loading-dots loading-sm"></span>
              </div>
            </div>
          </td>
          <td>
            <div class="flex gap-8">
              <button
                v-if="net.status === 'ready'"
                @click="toggleModify(net.id)"
                :disabled="net.group_role === 'member'"
                class="btn btn-primary btn-sm md:btn-md btn-outline rounded-lg"
              >
                <IconVue icon="material-symbols:edit" class="text-lg" />
                <p class="hidden md:inline">Edit</p>
              </button>
              <button
                v-if="net.status === 'ready' || net.status === 'unknown'"
                @click="preDeleteNet(net.id)"
                :disabled="net.group_role === 'member' || loading.is('net', net.id, 'delete')"
                class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
              >
                <span
                  v-if="loading.is('net', net.id, 'delete')"
                  class="loading loading-spinner loading-xs"
                ></span>
                <IconVue v-else icon="material-symbols:delete" class="text-lg" />
                <p class="hidden md:inline">Delete</p>
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete Net"
      positiveText="Delete Net"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteNet(netToDelete!)"
      @negative="cancelDeleteNet(netToDelete!)"
    >
      <p>Are you sure you want to delete this Net? This action cannot be undone.</p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
