<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import { useRouter } from 'vue-router'
import { login as _login } from '@/lib/api'
import type { Realm } from '@/types'

const router = useRouter()

const username = ref('')
const password = ref('')

const hideDrpdown = ref(true)
const realm = ref('Local')

const realms = ref<Realm[]>([])

function fetchRealms() {
  api
    .get('/login/realms')
    .then((res) => {
      realms.value = res.data as Realm[]
    })
    .catch((err) => {
      console.error('Failed to fetch realms:', err)
    })
}

async function login() {
  try {
    if (!username.value || !password.value) {
      console.error('Username and password are required')
      return
    }
    const realmID = realms.value.find((r) => r.name === realm.value)?.id
    if (!realmID) {
      console.error('Selected realm not found')
      return
    }
    await _login(username.value, password.value, realmID)
    router.push('/')
  } catch (error) {
    console.error('Login failed:', error)
  }
}

onMounted(() => {
  fetchRealms()
})
</script>

<template>
  <div class="grid h-dvh">
    <div class="w-96 place-self-center">
      <div class="text-center">Login for <b>Sasso!</b></div>
      <div class="mt-4">
        <input
          v-model="username"
          type="text"
          placeholder="Username"
          class="border p-2 rounded-lg mb-2 w-full"
        />
        <input
          v-model="password"
          type="password"
          placeholder="Password"
          class="border p-2 rounded-lg mb-2 w-full"
        />
      </div>
      <div class="flex flex-col items-center mb-2">
        <button
          @click="hideDrpdown = !hideDrpdown"
          class="text-gray-800 border border-gray-200 hover:bg-gray-100 focus:outline-none font-medium rounded-lg text-sm px-5 py-2.5 text-center inline-flex items-center"
          type="button"
        >
          {{ realm }}
          <svg
            class="w-2.5 h-2.5 ms-3"
            aria-hidden="true"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 10 6"
          >
            <path
              stroke="currentColor"
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="m1 1 4 4 4-4"
            />
          </svg>
        </button>

        <!-- Dropdown menu -->
        <div
          id="dropdown"
          :class="{ hidden: hideDrpdown }"
          class="z-10 bg-white divide-y divide-gray-100 rounded-lg shadow-sm w-44"
        >
          <ul class="py-2 text-sm text-gray-700" aria-labelledby="dropdownDefaultButton">
            <template v-for="r in realms" :key="r.id">
              <li>
                <a href="#" class="block px-4 py-2 hover:bg-gray-100" @click="realm = r.name">{{
                  r.name
                }}</a>
              </li>
            </template>
          </ul>
        </div>
      </div>
      <button class="bg-blue-400 p-2 rounded-lg w-full hover:bg-blue-300" @click="login()">
        Login
      </button>
    </div>
  </div>
</template>
