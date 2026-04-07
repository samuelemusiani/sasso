<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Realm } from '@/types'
import RealmsMultiplexer from '@/components/realms/RealmsMultiplexer.vue'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import ModalAlert from '@/components/ModalAlert.vue'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()
const loading = useLoadingStore()

const realms = ref<Realm[]>([])

const addingRealm = ref(false)
const addingType = ref('ldap')

function fetchRealms() {
  loading.start('realms', null, 'fetch')
  api
    .get('/admin/realms')
    .then((res) => {
      realms.value = res.data as Realm[]
    })
    .catch((err) => {
      console.error('Failed to fetch realms:', err)
      toastError('Failed to fetch realms: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('realms', null, 'fetch')
    })
}

function realmAdded() {
  addingRealm.value = false
  fetchRealms()
}

const showDeleteModal = ref(false)
const realmToDelete = ref<number | null>(null)

function preDeleteRealm(id: number) {
  realmToDelete.value = id
  showDeleteModal.value = true
  loading.start('realm', id, 'delete')
}

function deleteRealm(id: number) {
  api
    .delete(`/admin/realms/${id}`)
    .then(() => {
      // small optimization
      realms.value = realms.value.filter((realm) => realm.id !== id)
      fetchRealms()
    })
    .catch((err) => {
      console.error(`Failed to delete realm ${id}:`, err)
      toastError('Failed to delete realm')
    })
    .finally(() => {
      showDeleteModal.value = false
      realmToDelete.value = null
      loading.stop('realm', id, 'delete')
    })
}

function cancelDeleteRealm(id: number) {
  showDeleteModal.value = false
  realmToDelete.value = null
  loading.stop('realm', id, 'delete')
}

onMounted(() => {
  fetchRealms()
})
</script>

<template>
  <div class="p-2">
    <div class="flex justify-between">
      <AdminBreadcrumbs />
      <HelpButton />
    </div>
    <button class="btn btn-primary rounded-lg" @click="addingRealm = true" v-show="!addingRealm">
      Add LDAP Realm
    </button>

    <div v-if="loading.is('realms', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="mt-2 table w-full p-2" v-show="!addingRealm">
      <thead>
        <tr class="uppercase">
          <th class="">Name</th>
          <th class="">Description</th>
          <th class="">Type</th>
          <th class=""></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="realm in realms" :key="realm.id" class="odd:bg-base-100 even:bg-base-200">
          <td class="">{{ realm.name }}</td>
          <td class="">{{ realm.description }}</td>
          <td class="">{{ realm.type }}</td>
          <td class="">
            <div class="flex justify-start gap-2" v-show="realm.type != 'local'">
              <RouterLink
                class="btn btn-primary btn-sm md:btn-md rounded-lg"
                :to="`/admin/realms/${realm.id}`"
              >
                <IconVue icon="material-symbols:edit" class="text-lg" />
                <p class="hidden md:inline">Edit</p>
              </RouterLink>
              <button
                @click="preDeleteRealm(realm.id)"
                class="btn btn-error btn-sm md:btn-md rounded-lg"
                :disabled="loading.is('realm', realm.id, 'delete')"
              >
                <span
                  v-if="loading.is('realm', realm.id, 'delete')"
                  class="loading loading-spinner loading-xs"
                ></span>
                <IconVue v-else icon="material-symbols:delete" class="text-lg"></IconVue>
                <p class="hidden md:inline">Delete</p>
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>

    <RealmsMultiplexer
      v-show="addingRealm"
      :adding="addingRealm"
      :type="addingType"
      @realm-added="realmAdded"
    />

    <router-view class="mt-4" />

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete Realm"
      positiveText="Delete Realm"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteRealm(realmToDelete!)"
      @negative="cancelDeleteRealm(realmToDelete!)"
    >
      <p>Are you sure you want to delete this Realm? This action cannot be undone.</p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
