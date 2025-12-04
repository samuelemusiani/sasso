<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { TelegramBot } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { useToastService } from '@/composables/useToast'

const { error: toastError, success: toastSuccess } = useToastService()

const bots = ref<TelegramBot[]>([])
const name = ref('')
const notes = ref('')
const token = ref('')
const chat_id = ref('')

function fetchTelegramBots() {
  api
    .get('/notify/telegram')
    .then((res) => {
      const tmp = res.data.sort((a: TelegramBot, b: TelegramBot) => a.id - b.id)
      bots.value = tmp as TelegramBot[]
    })
    .catch((err) => {
      console.error('Failed to fetch Telegram Bots:', err)
    })
}

function requestTelegramBot() {
  api
    .post('/notify/telegram', {
      name: name.value,
      notes: notes.value,
      token: token.value,
      chat_id: chat_id.value,
    })
    .then(() => {
      fetchTelegramBots()
      name.value = ''
      notes.value = ''
      token.value = ''
      chat_id.value = ''
    })
    .catch((err) => {
      console.error('Failed to add Telegram Bot:', err)
    })
}

function deleteTelegramBot(id: number) {
  if (confirm('Are you sure you want to delete this Telegram Bot?')) {
    api
      .delete(`/notify/telegram/${id}`)
      .then(() => {
        fetchTelegramBots()
      })
      .catch((err) => {
        console.error('Failed to delete Telegram Bot:', err)
      })
  }
}

function testTelegramBot(id: number) {
  api
    .post(`/notify/telegram/${id}/test`)
    .then(() => {
      toastSuccess('Test notification sent successfully')
    })
    .catch((err) => {
      console.error('Failed to send test notification:', err)
      toastError('Failed to send test notification')
    })
}

function toggleEnableDisable(id: number, enabled: boolean) {
  api
    .patch(`/notify/telegram/${id}`, { enabled: enabled })
    .then(() => {
      fetchTelegramBots()
    })
    .catch((err) => {
      console.error('Failed to toggle enable/disable:', err)
    })
}

onMounted(() => {
  fetchTelegramBots()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <CreateNew title="Telegram Bot" :create="requestTelegramBot">
      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-2">
          <label for="name">Name</label>
          <input type="text" id="name" v-model="name" class="input w-48 rounded-lg border p-2" />
          <label for="token">Token</label>
          <input type="text" id="token" v-model="token" class="input w-48 rounded-lg border p-2" />
          <label for="chat_id">Chat ID</label>
          <input
            type="text"
            id="chat_id"
            v-model="chat_id"
            class="input w-48 rounded-lg border p-2"
          />
        </div>
        <div>
          <label for="notes">Notes</label>
          <textarea id="notes" v-model="notes" class="textarea w-full"></textarea>
        </div>
      </div>
    </CreateNew>

    <table class="table w-full table-auto">
      <thead>
        <tr>
          <th scope="col">Name</th>
          <th scope="col">Notes</th>
          <th scope="col">Chat ID</th>
          <th scope="col" class="">Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="bot in bots" :key="bot.id">
          <td class="min-w-40 text-lg font-semibold">{{ bot.name }}</td>
          <td class="">{{ bot.notes }}</td>
          <td class="">{{ bot.chat_id }}</td>
          <td class="flex gap-2">
            <button
              @click="testTelegramBot(bot.id)"
              class="btn btn-primary btn-sm md:btn-md btn-outline ml-2 rounded-lg"
            >
              <IconVue icon="material-symbols:experiment" class="text-lg" />
              <p>Test</p>
            </button>
            <button
              @click="toggleEnableDisable(bot.id, !bot.enabled)"
              :class="bot.enabled ? 'btn btn-warning' : 'btn btn-success'"
              class="btn btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue
                :icon="bot.enabled ? 'material-symbols:toggle-on' : 'material-symbols:toggle-off'"
                class="text-lg"
              />
              <p class="hidden md:inline">{{ bot.enabled ? 'Disable' : 'Enable' }}</p>
            </button>
            <button
              @click="deleteTelegramBot(bot.id)"
              class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
            >
              <IconVue icon="material-symbols:delete" class="text-lg" />
              <p class="hidden md:inline">Delete</p>
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
