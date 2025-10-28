<script setup lang="ts">
import { api } from '@/lib/api'
import { onMounted, ref, computed } from 'vue'
import type { Settings } from '@/types'
import { useToastService } from '@/composables/useToast'

const { error: toastError } = useToastService()

// Track which checkbox is loading: { "type_channel": true }
// We cannot user useLoadingStore because it's global for the VMs
const loadingStates = ref<Record<string, boolean>>({})

const isLoading = (type: string | number, channel: 'email' | 'telegram') => {
  return loadingStates.value[`${type}_${channel}`] ?? false
}

const settings = ref<Settings | null>(null)

function fetchSettings(): Promise<Settings> {
  return api
    .get('/settings')
    .then((response) => {
      return (settings.value = response.data as Settings)
    })
    .catch(() => {
      console.error('Failed to fetch settings')
      throw new Error('Failed to fetch settings')
    })
}

async function updateSetting() {
  try {
    await api.put('/settings', {
      ...settings.value,
    })

    // Now you can await it
    await fetchSettings()
  } catch (error) {
    toastError('Failed to update setting')
    console.error('Failed to update setting', error)
  }
}

type NotifySettings = {
  [key: string]: {
    email_notification: boolean
    telegram_notification: boolean
    toggleEmailNotification: () => void
    toggleTelegramNotification: () => void
  }
}

function getNotifySettingValue(
  settings: Settings,
  channel: 'mail' | 'telegram',
  key: string,
): boolean {
  const fullKey = `${channel}_${key}_notification` as keyof Settings
  return (settings[fullKey] as boolean) ?? false
}

function setNotifySettingValue(
  settings: Settings,
  channel: 'mail' | 'telegram',
  key: string,
  value: boolean,
): void {
  const fullKey = `${channel}_${key}_notification` as keyof Settings
  settings[fullKey] = value
}

const notifySettings = computed(() => {
  const notifySettings: NotifySettings = {}
  for (const key in settings.value) {
    if (key.endsWith('_notification')) {
      let nkey = key.replace('_notification', '')
      nkey = nkey.replace('mail_', '')
      nkey = nkey.replace('telegram_', '')

      notifySettings[nkey] = {
        email_notification: getNotifySettingValue(settings.value!, 'mail', nkey),
        telegram_notification: getNotifySettingValue(settings.value!, 'telegram', nkey),
        toggleEmailNotification: () => {
          loadingStates.value[`${nkey}_email`] = true
          setNotifySettingValue(
            settings.value!,
            'mail',
            nkey,
            !getNotifySettingValue(settings.value!, 'mail', nkey),
          )
          updateSetting().finally(() => {
            loadingStates.value[`${nkey}_email`] = false
          })
        },
        toggleTelegramNotification: () => {
          loadingStates.value[`${nkey}_telegram`] = true
          setNotifySettingValue(
            settings.value!,
            'telegram',
            nkey,
            !getNotifySettingValue(settings.value!, 'telegram', nkey),
          )
          updateSetting().finally(() => {
            loadingStates.value[`${nkey}_telegram`] = false
          })
        },
      }
    }
  }
  return notifySettings
})

onMounted(() => {
  fetchSettings()
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <h1 class="mb-2 text-2xl font-bold">Settings</h1>
    <table class="table w-full table-auto">
      <thead>
        <tr>
          <th class="">Notification settings</th>
          <th class="">Email</th>
          <th class="">Telegram</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(value, key) in notifySettings" :key="key">
          <td class="">{{ key }}</td>
          <td class="">
            <input
              v-if="!isLoading(key, 'email')"
              type="checkbox"
              class="checkbox checkbox-primary"
              :checked="value.email_notification"
              @change="value.toggleEmailNotification()"
            />

            <span v-else class="loading loading-spinner text-primary loading-md"></span>
          </td>
          <td class="">
            <input
              v-if="!isLoading(key, 'telegram')"
              type="checkbox"
              class="checkbox checkbox-primary"
              :checked="value.telegram_notification"
              @change="value.toggleTelegramNotification()"
            />
            <span v-else class="loading loading-spinner text-primary loading-md"></span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
