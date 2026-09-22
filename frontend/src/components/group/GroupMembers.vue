<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import type { Group, GroupMember, GroupResource, GroupInvite } from '@/types'
import { useLoadingStore } from '@/stores/loading'
import { useToastService } from '@/composables/useToast'
import CreateNew from '@/components/CreateNew.vue'
import ModalAlert from '@/components/ModalAlert.vue'

const $props = defineProps<{
  group: Group
  me: GroupMember
}>()

const $emit = defineEmits(['fetch-members', 'fetch-group'])

const username = ref('')
const role = ref('member')
const error = ref('')

const { error: toastError, success: toastSuccess } = useToastService()
const loading = useLoadingStore()

const invitations = ref<GroupInvite[]>([])

function getResourcesForUser(userId: number): GroupResource | undefined {
  return $props.group?.resources?.find((r) => r.user_id === userId)
}

function inviteUser() {
  if (!username.value) {
    error.value = 'Username is required to invite a user.'
    return false
  }

  return api
    .post(`/groups/${$props.group.id}/invites`, {
      username: username.value,
      role: role.value,
    })
    .then(() => {
      username.value = ''
      fetchInvitations()
      toastSuccess('User invited successfully.')

      return true
    })
    .catch((err) => {
      console.error('Failed to invite user:', err)
      error.value = `Failed to invite user: ${err.response?.data}`

      return false
    })
}

const showDeleteModal = ref(false)
const deleteTargetId = ref<number | null>(null) // e.g. $props.group.id for group deletion or inviteId for invitation revocation
const deleteModalObject = ref('') // e.g. 'Group' or 'Invitation'
const titleDeleteModal = computed(() => {
  switch (deleteModalObject.value) {
    case 'Invitation':
      return 'Revoke Invitation'
    case 'Member':
      return 'Remove Member'
    default:
      return ''
  }
})

const positiveTextDeleteModal = computed(() => {
  switch (deleteModalObject.value) {
    case 'Invitation':
      return 'Revoke invitation'
    case 'Member':
      return 'Remove member'
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
    case 'Invitation':
      return 'Are you sure you want to revoke this invitation?'
    case 'Member':
      return 'Are you sure you want to remove this member?'
    default:
      return ''
  }
})

function preDeleteMember(id: number) {
  deleteModalObject.value = 'Member'
  deleteTargetId.value = id
  showDeleteModal.value = true
  loading.start('groupMember', id, 'delete')
}

function deleteMember(id: number) {
  api
    .delete(`/groups/${$props.group.id}/members/${id}`)
    .then(() => {
      $emit('fetch-members')
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

function cancelDeleteMember(id: number) {
  resetDeleteModal()
  loading.stop('groupMember', id, 'delete')
}

function preRevokeUserInvite(id: number) {
  deleteModalObject.value = 'Invitation'
  deleteTargetId.value = id
  showDeleteModal.value = true
  loading.start('groupInvite', id, 'delete')
}

function cancelRevokeUserInvite(id: number) {
  resetDeleteModal()
  loading.stop('groupInvite', id, 'delete')
}

function revokeUserInvite(id: number) {
  api
    .delete(`/groups/${$props.group.id}/invites/${id}`)
    .then(() => {
      // small optimization
      invitations.value = invitations.value.filter((invite) => invite.id !== id)
      fetchInvitations()
    })
    .catch((err) => {
      console.error('Failed to revoke invitation:', err)
      toastError(`Failed to revoke invitation. ${err.response?.data}`)
    })
    .finally(() => {
      resetDeleteModal()
      loading.stop('groupInvite', id, 'delete')
    })
}

function deleteModalPositiveFunc(id: number) {
  switch (deleteModalObject.value) {
    case 'Invitation':
      revokeUserInvite(id)
      break
    case 'Member':
      deleteMember(id)
      break
    default:
      console.error('Unknown delete object:', deleteModalObject.value)
  }
}

function deleteModalNegativeFunc(id: number) {
  switch (deleteModalObject.value) {
    case 'Invitation':
      cancelRevokeUserInvite(id)
      break
    case 'Member':
      cancelDeleteMember(id)
      break
    default:
      console.error('Unknown delete object:', deleteModalObject.value)
  }
}

function fetchInvitations() {
  api
    .get(`/groups/${$props.group.id}/invites`)
    .then((res) => {
      const tmp = res.data.sort((a: GroupInvite, b: GroupInvite) => a.id - b.id)
      invitations.value = tmp as GroupInvite[]
    })
    .catch((err) => {
      console.error('Failed to fetch Invitations:', err)
      toastError(`Failed to fetch Invitations. ${err.response?.data}`)
    })
}

onMounted(() => {
  fetchInvitations()
})
</script>

<template>
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
              <th>Actions</th>
            </tr>
          </thead>
          <tbody v-if="group?.members">
            <tr
              v-for="member in group.members"
              :key="member.user_id"
              class="odd:bg-base-100 even:bg-base-200"
            >
              <td>
                {{ member.username }}<span class="opacity-60">@{{ member.realm_name }}</span>
              </td>
              <td>{{ member.role }}</td>
              <td>{{ getResourcesForUser(member.user_id)?.cores || 0 }}</td>
              <td>{{ getResourcesForUser(member.user_id)?.ram || 0 }}</td>
              <td>{{ getResourcesForUser(member.user_id)?.disk || 0 }}</td>
              <td>{{ getResourcesForUser(member.user_id)?.nets || 0 }}</td>
              <td>
                <button
                  v-show="me && me.role == 'owner' && member.user_id != me.user_id"
                  @click="preDeleteMember(member.user_id)"
                  class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
                  :disabled="loading.is('groupMember', member.user_id, 'delete')"
                >
                  <span
                    v-if="loading.is('groupMember', member.user_id, 'delete')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:delete" class="text-lg"></IconVue>
                  <p class="hidden md:inline">Remove</p>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <div class="divider my-4"></div>

  <div class="flex flex-col gap-2">
    <h2 class="mb-2 text-xl font-semibold">Pending Invitations</h2>

    <CreateNew title="Invitation" :create="inviteUser" :error="error" :close-on-create="true">
      <div class="flex flex-col gap-2">
        <div class="grid grid-cols-2 gap-8">
          <div>
            <label for="username">Username</label>
            <input
              type="text"
              id="username"
              v-model="username"
              class="input w-full rounded-lg border p-2"
              placeholder="Username to invite"
            />
          </div>
          <div>
            <label for="role">Role</label>
            <select id="role" v-model="role" class="input w-full rounded-lg border p-2">
              <option value="member">Member</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        </div>
      </div>
    </CreateNew>

    <div>
      <template v-if="invitations.length === 0">
        <p>No pending invitations.</p>
      </template>
      <div v-else class="overflow-x-auto">
        <table class="table w-full">
          <thead>
            <tr>
              <th>Username</th>
              <th>Role</th>
              <th>State</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="invite in invitations"
              :key="invite.id"
              class="odd:bg-base-100 even:bg-base-200"
            >
              <td>
                {{ invite.username }}<span class="opacity-60">@{{ invite.realm_name }}</span>
              </td>
              <td>{{ invite.role }}</td>
              <td>{{ invite.state }}</td>
              <td>
                <button
                  @click="preRevokeUserInvite(invite.id)"
                  class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
                  :disabled="loading.is('groupInvite', invite.id, 'delete')"
                >
                  <span
                    v-if="loading.is('groupInvite', invite.id, 'delete')"
                    class="loading loading-spinner loading-xs"
                  ></span>
                  <IconVue v-else icon="material-symbols:undo" class="text-lg"></IconVue>
                  <p class="hidden md:inline">Revoke</p>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
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
