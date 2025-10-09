<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'

interface Props {
  code?: number | string
  message?: string
  buttons?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  code: 404,
  message: '',
  buttons: true
})

const router = useRouter()

const errorCode = computed<number | string>(() => props.code)

const errorTitle = computed<string>(() => {
  const titles: Record<number, string> = {
    400: 'Bad Request',
    401: 'Unauthorized',
    403: 'Forbidden',
    404: 'Page Not Found',
    500: 'Internal Server Error',
    503: 'Service Unavailable'
  }
  return titles[Number(errorCode.value)] || 'Oops! Something Went Wrong'
})

const errorMessage = computed<string>(() => {
  if (props.message) return props.message

  const messages: Record<number, string> = {
    400: 'The request could not be understood by the server.',
    401: 'You need to be authenticated to access this page.',
    403: 'You don\'t have permission to access this resource.',
    404: 'The page you are looking for doesn\'t exist or has been moved.',
    500: 'Something went wrong on our end. Please try again later.',
    503: 'The service is temporarily unavailable. Please try again later.'
  }
  return messages[Number(errorCode.value)] || 'An unexpected error occurred.'
})

const goBack = (): void => {
  router.go(-1)
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center w-full">
    <div class="card w-full max-w-2xl shadow-2xl">
      <div class="card-body items-center text-center">
        <img src="/sasso-error.png" alt="sasso-error" class="w-64" />

        <div class="card-title text-3xl mb-4">
          <h1
            class="text-5xl font-black bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent leading-none">
            {{ errorCode }}
          </h1>
          {{ errorTitle }}
        </div>

        <p class="text-base-content/70 text-lg mb-2">
          {{ errorMessage }}
        </p>

        <div v-show="props.buttons" class="divider"></div>

        <div v-show="props.buttons" class="card-actions flex-col sm:flex-row gap-3 w-full sm:w-auto">
          <router-link to="/" class="btn btn-primary btn-lg gap-2 rounded-lg">
            <IconVue icon="mdi:home" class="w-5 h-5" />
            Go Home
          </router-link>

          <button @click="goBack" class="btn btn-outline btn-lg gap-2 rounded-lg">
            <IconVue icon="mdi:arrow-left" class="w-5 h-5" />
            Go Back
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
