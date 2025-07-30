<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login as _login } from '@/lib/api'

const router = useRouter()

const username = ref('')
const password = ref('')

async function login() {
  try {
    await _login(username.value, password.value)
    router.push('/vm')
  } catch (error) {
    console.error('Login failed:', error)
  }
}
</script>

<template>
  <div class="grid h-dvh">
    <div class="w-96 place-self-center">
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
      <button class="bg-blue-400 p-2 rounded-lg w-full hover:bg-blue-300" @click="login()">
        Login
      </button>
    </div>
  </div>
</template>
