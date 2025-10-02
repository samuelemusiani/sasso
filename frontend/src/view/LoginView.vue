<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import { useRouter } from 'vue-router'
import { login as _login } from '@/lib/api'
import type { Realm } from '@/types'

const router = useRouter()

const username = ref('')
const password = ref('')

const showPassword = ref(false)
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

<!-- TODO: save login preference -->
<template>
  <div class="grid h-screen place-items-center">
    <div class="flex flex-col items-center gap-2">
      <div class="text-center flex items-center gap-2">
        Login into <img src="/public/sasso.png" class="h-20" />
      </div>
      <div class="w-full">
        <legend class="label mb-1">Username</legend>
        <label class="input validator rounded-lg">
          <IconVue icon="material-symbols:person" class="h-[1em] opacity-50 text-lg" />
          <input
            type="text"
            required
            placeholder="Username"
            pattern="[A-Za-z][A-Za-z0-9\-]*"
            minlength="3"
            maxlength="30"
            title="Only letters, numbers or dash"
          />
        </label>
      </div>

      <div class="w-full">
        <legend class="label mb-1">Password</legend>
        <label class="input rounded-lg">
          <IconVue icon="material-symbols:lock" class="h-[1em] opacity-50 text-lg" />
          <input
            required
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            placeholder="Password"
            class="grow"
          />
          <button
            type="button"
            @click="showPassword = !showPassword"
            class="btn btn-ghost btn-circle w-auto h-auto hover:bg-transparent hover:border-0"
          >
            <IconVue
              :icon="
                showPassword ? 'material-symbols:visibility-off' : 'material-symbols:visibility'
              "
              class="text-lg text-base-content/50"
            />
          </button>
        </label>
      </div>

      <div v-if="realms.length === 0" class="text-error">
        There's a connection error, please try to refresh
      </div>
      <fieldset v-else class="my-2 w-full">
        <legend class="label mb-1">Realms</legend>
        <select class="flex flex-col items-center select rounded-lg">
          <template v-for="r in realms" :key="r.id">
            <option class="block px-4 py-2" @click="realm = r.name">
              {{ r.name }}
            </option>
          </template>
        </select>
      </fieldset>
      <button class="btn btn-primary p-2 rounded-lg w-full" @click="login()">Login</button>
    </div>
  </div>
  <p class="text-center text-base-content/50 absolute inset-x-0 bottom-8">
    by
    <a href="https://students.cs.unibo.it" class="text-primary"
      ><img src="/ADMStaff.svg" class="opacity-70 h-8 inline" alt="ADMStaff"
    /></a>
  </p>
  <!-- 
    <p class="text-center text-base-content/50 absolute inset-x-0 bottom-8 ">Developed by <a
      href="https://students.cs.unibo.it" class="text-primary">ADMStaff</a></p> -->
</template>
