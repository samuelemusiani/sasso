<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import { useRouter } from 'vue-router'
import { login as _login } from '@/lib/api'
import type { Realm } from '@/types'
import type { AxiosError } from 'axios'

const router = useRouter()

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const realm = ref('Local')
const realms = ref<Realm[]>([])

const errorMessage = ref('')

function fetchRealms() {
  api
    .get('/login/realms')
    .then((res) => {
      realms.value = res.data
      const savedRealm = localStorage.getItem('realm')
      if (savedRealm && realms.value.some((r) => r.name === savedRealm)) {
        realm.value = savedRealm
      }
    })
    .catch((err) => {
      console.error('Failed to fetch realms:', err)
    })
}

async function login() {
  try {
    if (!username.value || !password.value) {
      console.error('Username and password are required')
      errorMessage.value = 'Username and password are required'
      return
    }
    const realmID = realms.value.find((r) => r.name === realm.value)?.id
    if (!realmID) {
      console.error('Selected realm not found')
      return
    }
    localStorage.setItem('realm', realm.value)
    await _login(username.value, password.value, realmID)
    router.push('/')
  } catch (error) {
    const axiosError = error as AxiosError
    console.error('Login failed:', error)
    if (axiosError.status === 401) {
      errorMessage.value = 'Invalid username or password'
    } else if (axiosError.status === 500) {
      errorMessage.value = "There's a connection error, please try to refresh"
    } else {
      errorMessage.value = 'An error occurred during login'
    }
  }
}

onMounted(() => {
  fetchRealms()
})
</script>

<template>
  <div class="flex-1 overflow-auto">
    <div class="grid h-screen place-items-center">
      <div class="flex flex-col items-center gap-2">
        <div class="flex items-center gap-2 text-center">
          Login into <img src="/sasso.png" class="h-20" />
        </div>
        <div class="w-full">
          <legend class="label mb-1">Username</legend>
          <label class="input validator rounded-lg">
            <IconVue icon="material-symbols:person" class="h-[1em] text-lg opacity-50" />
            <input
              type="text"
              v-model="username"
              required
              placeholder="Username"
              pattern="[A-Za-z][A-Za-z0-9\-]*"
              minlength="3"
              maxlength="30"
              title="Only letters, numbers or dash"
              @keyup.enter="login"
            />
          </label>
        </div>

        <div class="w-full">
          <legend class="label mb-1">Password</legend>
          <label class="input rounded-lg">
            <IconVue icon="material-symbols:lock" class="h-[1em] text-lg opacity-50" />
            <input
              required
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Password"
              class="grow"
              @keyup.enter="login"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              class="btn btn-ghost btn-circle h-auto w-auto hover:border-0 hover:bg-transparent"
            >
              <IconVue
                :icon="
                  showPassword ? 'material-symbols:visibility-off' : 'material-symbols:visibility'
                "
                class="text-base-content/50 text-lg"
              />
            </button>
          </label>
        </div>

        <div v-if="errorMessage" class="text-error">
          {{ errorMessage }}
        </div>
        <fieldset class="my-2 w-full">
          <legend class="label mb-1">Realms</legend>
          <select v-model="realm" class="select w-full rounded-lg">
            <option v-for="r in realms" :key="r.id" :value="r.name">
              {{ r.name }}
            </option>
          </select>
        </fieldset>
        <button class="btn btn-primary w-full rounded-lg p-2" @click="login()">Login</button>
      </div>
    </div>
    <p class="text-base-content/50 absolute inset-x-0 bottom-8 text-center">
      by
      <a href="https://students.cs.unibo.it" class="text-primary">
        <img src="/ADMStaff.svg" class="inline h-8 opacity-70" alt="ADMStaff" />
      </a>
    </p>
  </div>
</template>
