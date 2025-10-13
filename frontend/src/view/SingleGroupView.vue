<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Group, GroupInvite, GroupMember } from '@/types'
import { api } from '@/lib/api'
import AdminBreadcrumbs from '@/components/AdminBreadcrumbs.vue'
import CreateNew from '@/components/CreateNew.vue'

const group = ref<Group | null>(null)

const route = useRoute()
const router = useRouter()
const groupId = Number(route.params.id)

const username = ref('')
const role = ref('member')

const invitations = ref<GroupInvite[]>([])
// const members = ref<GroupMember[]>([])

const me = ref<GroupMember | null>(null)

function fetchGroup() {
  api
    .get(`/groups/${groupId}`)
    .then((res) => {
      group.value = res.data as Group
    })
    .catch((err) => {
      console.error('Failed to fetch Group:', err)
    })
}

function inviteUser() {
  if (!username.value) {
    alert('Please enter a username.')
    return
  }
  api
    .post(`/groups/${groupId}/invites`, {
      username: username.value,
      role: role.value,
    })
    .then(() => {
      alert(`Invitation sent to ${username.value}`)
      username.value = ''
    })
    .catch((err) => {
      console.error('Failed to invite user:', err)
      alert('Failed to send invitation. Please try again.')
    })
}

function fetchInvitations() {
  api
    .get(`/groups/${groupId}/invites`)
    .then((res) => {
      const tmp = res.data.sort((a: GroupInvite, b: GroupInvite) => a.id - b.id)
      invitations.value = tmp as GroupInvite[]
    })
    .catch((err) => {
      console.error('Failed to fetch Invitations:', err)
    })
}

function revokeUserInvite(id: number) {
  if (confirm('Are you sure you want to revoke this invitation?')) {
    api
      .delete(`/groups/${groupId}/invites/${id}`)
      .then(() => {
        fetchInvitations()
      })
      .catch((err) => {
        console.error('Failed to revoke invitation:', err)
      })
  }
}

function fetchMembers() {
  api
    .get(`/groups/${groupId}/members`)
    .then((res) => {
      const tmp = res.data.sort((a: GroupMember, b: GroupMember) => a.user_id - b.user_id)
      if (group.value) {
        group.value.members = tmp as GroupMember[]
      }
    })
    .catch((err) => {
      console.error('Failed to fetch Members:', err)
    })
}

function fetchMe() {
  api
    .get(`/groups/${groupId}/members/me`)
    .then((res) => {
      me.value = res.data as GroupMember
    })
    .catch((err) => {
      console.error('Failed to fetch current user membership:', err)
    })
}

function deleteMember(id: number) {
  let leave_me = false
  let msg = ''
  let id_path = id.toString()
  if (id === me.value?.user_id) {
    msg = 'leave the group'
    id_path = 'me'
    leave_me = true
  } else {
    msg = 'remove this member'
  }
  if (confirm('Are you sure you want to ' + msg + '?')) {
    api
      .delete(`/groups/${groupId}/members/${id_path}`)
      .then(() => {
        if (leave_me) {
          router.push('/admin/groups')
          return
        }
        fetchMembers()
      })
      .catch((err) => {
        console.error('Failed to remove member:', err)
      })
  }
}

function deleteGroup(id: number) {
  if (confirm('Are you sure you want to delete this Group?')) {
    api
      .delete(`/groups/${id}`)
      .then(() => {
        router.push('/admin/groups')
      })
      .catch((err) => {
        console.error('Failed to delete Group:', err)
      })
  }
}

onMounted(() => {
  fetchMe()
  fetchGroup()
  // fetchMembers()
  fetchInvitations()
})
</script>

<template>
  <div class="p-2">
    <AdminBreadcrumbs />

    <CreateNew title="Invitation" :create="inviteUser">
      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-2">
          <label for="username">Username</label>
          <input
            type="text"
            id="username"
            v-model="username"
            class="input w-48 rounded-lg border p-2"
          />
          <label for="role">Role</label>
          <select id="role" v-model="role" class="input w-48 rounded-lg border p-2">
            <option value="member">Member</option>
            <option value="admin">Admin</option>
          </select>
        </div>
      </div>
    </CreateNew>

    <div>
      <button
        v-if="me && me.role != 'owner'"
        @click="deleteMember(me.user_id)"
        class="btn btn-error rounded-lg"
      >
        Leave Group
      </button>
      <button v-else @click="deleteGroup(groupId)" class="btn btn-error rounded-lg">
        Delete Group
      </button>
    </div>

    <div v-if="group" class="rounded-lg p-4 shadow">
      <h2 class="mb-2 text-xl font-semibold">{{ group.name }}</h2>
      <p class="mb-4 text-gray-600">{{ group.description }}</p>
    </div>

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
                <th>Actions</th>
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
                <td>
                  <button
                    v-show="me && me.role == 'owner' && member.user_id != me.user_id"
                    @click="deleteMember(member.user_id)"
                    class="btn btn-sm btn-error"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="divider my-4"></div>

    <div>
      <h2 class="mb-2 text-xl font-semibold">Pending Invitations</h2>
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
                <td>{{ invite.username }}</td>
                <td>{{ invite.role }}</td>
                <td>{{ invite.state }}</td>
                <td>
                  <button @click="revokeUserInvite(invite.id)" class="btn btn-sm btn-error">
                    Revoke
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
