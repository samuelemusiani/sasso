<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { TelegramBot } from '@/types'
import { api } from '@/lib/api'
import CreateNew from '@/components/CreateNew.vue'
import { useToastService } from '@/composables/useToast'
import { useLoadingStore } from '@/stores/loading'
import ModalAlert from '@/components/ModalAlert.vue'
import { getPageIcon } from '@/const'

const { error: toastError, success: toastSuccess } = useToastService()
const loading = useLoadingStore()

const bots = ref<TelegramBot[]>([])
const name = ref('')
const notes = ref('')
const token = ref('')
const chat_id = ref('')
const error = ref('')

function fetchTelegramBots() {
  loading.start('telegramBots', null, 'fetch')
  api
    .get('/notify/telegram')
    .then((res) => {
      const tmp = res.data.sort((a: TelegramBot, b: TelegramBot) => a.id - b.id)
      bots.value = tmp as TelegramBot[]
    })
    .catch((err) => {
      console.error('Failed to fetch Telegram Bots:', err)
      toastError('Failed to fetch Telegram Bots: ' + err.response.data)
    })
    .finally(() => {
      loading.stop('telegramBots', null, 'fetch')
    })
}

function requestTelegramBot() {
  return api
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

      return true
    })
    .catch((err) => {
      console.error('Failed to add Telegram Bot:', err)
      error.value = 'Failed to add Telegram Bot: ' + err.response.data
      return false
    })
}

const showDeleteModal = ref(false)
const botToDelete = ref<number | null>(null)

function preDeleteTelegramBot(id: number) {
  botToDelete.value = id
  showDeleteModal.value = true
  loading.start('telegramBot', id, 'delete')
}

function deleteTelegramBot(id: number) {
  api
    .delete(`/notify/telegram/${id}`)
    .then(() => {
      fetchTelegramBots()
    })
    .catch((err) => {
      console.error('Failed to delete Telegram Bot:', err)
      toastError('Failed to delete Telegram Bot')
    })
    .finally(() => {
      botToDelete.value = null
      showDeleteModal.value = false
      loading.stop('telegramBot', id, 'delete')
    })
}

function cancelDeleteTelegramBot(id: number) {
  botToDelete.value = null
  showDeleteModal.value = false
  loading.stop('telegramBot', id, 'delete')
}

function testTelegramBot(id: number) {
  loading.start('telegramBot', id, 'test')
  api
    .post(`/notify/telegram/${id}/test`)
    .then(() => {
      toastSuccess('Test notification sent successfully')
    })
    .catch((err) => {
      console.error('Failed to send test notification:', err)
      toastError('Failed to send test notification')
    })
    .finally(() => {
      loading.stop('telegramBot', id, 'test')
    })
}

function toggleEnableDisable(id: number, enabled: boolean) {
  loading.start('telegramBot', id, 'toggle')
  api
    .patch(`/notify/telegram/${id}`, { enabled: enabled })
    .then(() => {
      fetchTelegramBots()
    })
    .catch((err) => {
      console.error('Failed to toggle enable/disable:', err)
      toastError('Failed to toggle enable/disable')
    })
    .finally(() => {
      loading.stop('telegramBot', id, 'toggle')
    })
}

onMounted(() => {
  fetchTelegramBots()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <div class="flex justify-between">
      <h1 class="flex items-center gap-2 text-3xl font-bold">
        <IconVue class="text-primary" :icon="getPageIcon('telegram')"></IconVue>Telegram Bots
      </h1>
      <HelpButton />
    </div>

    <CreateNew
      title="Telegram Bot"
      :create="requestTelegramBot"
      :error="error"
      :close-on-create="true"
    >
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

    <div v-if="loading.is('telegramBots', null, 'fetch')" class="grid h-64">
      <span class="loading loading-spinner loading-lg text-primary place-self-center"></span>
    </div>

    <table v-else class="table w-full table-auto">
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
              :disabled="loading.is('telegramBot', bot.id, 'test')"
            >
              <span
                v-if="loading.is('telegramBot', bot.id, 'test')"
                class="loading loading-spinner loading-xs"
              ></span>
              <IconVue v-else icon="material-symbols:experiment" class="text-lg" />
              <p>Test</p>
            </button>
            <button
              @click="toggleEnableDisable(bot.id, !bot.enabled)"
              :class="bot.enabled ? 'btn btn-warning' : 'btn btn-success'"
              class="btn btn-sm md:btn-md btn-outline w-30 rounded-lg"
              :disabled="loading.is('telegramBot', bot.id, 'toggle')"
            >
              <span
                v-if="loading.is('telegramBot', bot.id, 'toggle')"
                class="loading loading-spinner loading-xs"
              ></span>
              <IconVue
                v-else
                :icon="bot.enabled ? 'material-symbols:toggle-on' : 'material-symbols:toggle-off'"
                class="text-lg"
              />
              <p class="hidden md:inline">{{ bot.enabled ? 'Disable' : 'Enable' }}</p>
            </button>
            <button
              @click="preDeleteTelegramBot(bot.id)"
              class="btn btn-error btn-sm md:btn-md btn-outline rounded-lg"
              :disabled="loading.is('telegramBot', bot.id, 'delete')"
            >
              <span
                v-if="loading.is('telegramBot', bot.id, 'delete')"
                class="loading loading-spinner loading-xs"
              ></span>
              <IconVue v-else icon="material-symbols:delete" class="text-lg" />
              <p class="hidden md:inline">Delete</p>
            </button>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Delete modal -->
    <ModalAlert
      :model-value="showDeleteModal"
      title="Delete Bot"
      positiveText="Delete Bot"
      negativeText="Cancel action"
      positiveBtnClass="btn-error"
      @positive="deleteTelegramBot(botToDelete!)"
      @negative="cancelDeleteTelegramBot(botToDelete!)"
    >
      <p>Are you sure you want to delete this Bot key? This action cannot be undone.</p>
    </ModalAlert>
    <!-- End of Delete modal -->
  </div>
</template>
