<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import type { Group, GroupMember } from '@/types'
import { api } from '@/lib/api'
import { useToastService } from '@/composables/useToast'
import ModalAlert from '@/components/ModalAlert.vue'
import { useLoadingStore } from '@/stores/loading'

const $props = defineProps<{
  group: Group
  me: GroupMember
}>()

const $emit = defineEmits(['fetch-group'])

const { error: toastError, success: toastSuccess } = useToastService()
const loading = useLoadingStore()

const router = useRouter()

const editing = ref(false)
const error = ref('')

const groupName = ref($props.group?.name || '')
const groupDescription = ref($props.group?.description || '')

watch($props.group, (newGroup) => {
  if (newGroup) {
    groupName.value = newGroup.name
    groupDescription.value = newGroup.description
  }
})

const showDeleteModal = ref(false)
const deleteTargetId = ref<number | null>(null) // e.g. $props.group.id for group deletion or inviteId for invitation revocation
const deleteModalObject = ref('') // e.g. 'Group' or 'Invitation'
const titleDeleteModal = computed(() => {
  switch (deleteModalObject.value) {
    case 'Group':
      return 'Delete Group'
    case 'Me':
      return 'Leave Group'
    default:
      return ''
  }
})

const positiveTextDeleteModal = computed(() => {
  switch (deleteModalObject.value) {
    case 'Group':
      return 'Delete group'
    case 'Me':
      return 'Leave group'
    default:
      return ''
  }
})

function resetDeleteModal() {
  showDeleteModal.value = false
  new Promise((resolve) => setTimeout(resolve, 300)).then(() => {
    // wait for modal close animation to finish before resetting the object and id
    deleteModalObject.value = ''
    deleteTargetId.value = null
  })
}

const bodyDeleteModal = computed(() => {
  switch (deleteModalObject.value) {
    case 'Group':
      return 'Are you sure you want to delete this group? This action cannot be undone.'
    case 'Me':
      return 'Are you sure you want to leave this group?'
    default:
      return ''
  }
})

function cancelExitGroup(id: number) {
  resetDeleteModal()
  loading.stop('groupMember', id, 'delete')
}

function preDeleteGroup(id: number) {
  deleteModalObject.value = 'Group'
  deleteTargetId.value = id
  showDeleteModal.value = true
  loading.start('group', $props.group.id, 'delete')
}

function deleteGroup(id: number) {
  api
    .delete(`/groups/${id}`)
    .then(() => {
      toastSuccess('Group deleted successfully.')
      router.push('/group')
    })
    .catch((err) => {
      console.error('Failed to delete Group:', err)
      toastError(`Failed to delete Group. ${err.response?.data}`)
    })
    .finally(() => {
      resetDeleteModal()
      loading.stop('group', id, 'delete')
    })
}

function cancelDeleteGroup(id: number) {
  resetDeleteModal()
  loading.stop('group', id, 'delete')
}

function deleteModalPositiveFunc(id: number) {
  switch (deleteModalObject.value) {
    case 'Group':
      deleteGroup(id)
      break
    case 'Me':
      exitGroup(id)
      break
    default:
      console.error('Unknown delete object:', deleteModalObject.value)
  }
}

function preExitGroup(id: number) {
  deleteModalObject.value = 'Me'
  deleteTargetId.value = id
  showDeleteModal.value = true
  loading.start('groupMember', id, 'delete')
}

async function exitGroup(id: number) {
  return api
    .delete(`/groups/${$props.group.id}/members/me`)
    .then(() => {
      router.replace('/group')
      return
    })
    .catch((err) => {
      console.error('Failed to remove member:', err)
      toastError(`Failed to remove member. ${err.response?.data}`)
    })
    .finally(() => {
      resetDeleteModal()
      loading.stop('groupMember', id, 'delete')
    })
}

function deleteModalNegativeFunc(id: number) {
  switch (deleteModalObject.value) {
    case 'Group':
      cancelDeleteGroup(id)
      break
    case 'Me':
      cancelExitGroup(id)
      break
    default:
      console.error('Unknown delete object:', deleteModalObject.value)
  }
}

async function updateGroup() {
  return api
    .put(`/groups/${$props.group.id}`, {
      name: groupName.value,
      description: groupDescription.value,
    })
    .then(() => {
      $emit('fetch-group')
      toastSuccess('Group updated successfully.')
      editing.value = false
      return true
    })
    .catch((err) => {
      console.error('Failed to update Group:', err)
      error.value = `Failed to update Group. ${err.response?.data}`
      return false
    })
}
</script>

<template>
  <div class="">
    <div v-if="group" class="flex flex-col gap-4 rounded-lg">
      <div class="flex items-center justify-between">
        <h2 v-if="!editing" class="text-xl font-bold">{{ group.name }}</h2>
        <div v-else>
          <label for="groupName" class="">Group Name</label>
          <input
            required
            type="text"
            v-model="groupName"
            class="input w-full max-w-xs rounded-lg border p-2"
            placeholder="Group Name"
          />
        </div>
        <button
          v-show="$props.me && $props.me.role == 'owner'"
          class="btn btn-outline rounded-lg"
          :class="editing ? 'btn-error' : 'btn-primary'"
          @click="editing = !editing"
        >
          <IconVue :icon="editing ? 'mdi:remove' : 'material-symbols:edit'" class="text-lg" />
          <p class="hidden md:inline">{{ editing ? 'Cancel' : 'Edit' }}</p>
        </button>
      </div>
      <p v-if="!editing" class="whitespace-pre-wrap text-gray-600">{{ group.description }}</p>
      <div v-else>
        <label for="groupDescription" class="">Group Description</label>
        <textarea
          class="textarea w-full"
          placeholder="Group Description"
          v-model="groupDescription"
        ></textarea>
      </div>
      <div v-if="editing" class="flex gap-2">
        <button
          @click="updateGroup()"
          class="btn btn-success w-full rounded-lg"
          :disabled="loading.is('group', $props.group.id, 'update')"
        >
          <span
            v-if="loading.is('group', $props.group.id, 'update')"
            class="loading loading-spinner loading-xs"
          ></span>
          <IconVue v-else icon="material-symbols:save" class="text-lg"></IconVue>
          <p class="hidden md:inline">Save</p>
        </button>
      </div>
    </div>

    <div class="divider text-error my-4 font-bold">Danger Zone</div>

    <div class="my-2 flex gap-2">
      <button
        v-if="$props.me && $props.me.role != 'owner'"
        @click="preExitGroup($props.me.user_id)"
        class="btn btn-error btn-outline w-70 rounded-lg"
        :disabled="loading.is('groupMember', $props.me.user_id, 'delete')"
      >
        <span
          v-if="loading.is('groupMember', $props.me.user_id, 'delete')"
          class="loading loading-spinner loading-xs"
        ></span>
        <IconVue v-else icon="pepicons-pencil:leave" class="text-lg"></IconVue>
        <p class="hidden md:inline">Leave Group</p>
      </button>
      <button
        v-else
        @click="preDeleteGroup($props.group.id)"
        class="btn btn-error btn-sm md:btn-md btn-outline w-70 rounded-lg"
        :disabled="loading.is('group', $props.group.id, 'delete')"
      >
        <span
          v-if="loading.is('group', $props.group.id, 'delete')"
          class="loading loading-spinner loading-xs"
        ></span>
        <IconVue v-else icon="material-symbols:delete" class="text-lg"></IconVue>
        <p class="hidden md:inline">Delete</p>
      </button>
    </div>

    <!-- Delete modal -->

    <ModalAlert
      :model-value="showDeleteModal"
      :title="titleDeleteModal"
      :positiveText="positiveTextDeleteModal"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteModalPositiveFunc(deleteTargetId!)"
      @negative="deleteModalNegativeFunc(deleteTargetId!)"
    >
      <p>
        {{ bodyDeleteModal }}
      </p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
