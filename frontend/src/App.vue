<script setup lang="ts">
import { RouterView } from 'vue-router'
import { onMounted } from 'vue'
import ToastContainer from '@/components/ToastContainer.vue'
import { initToastService } from '@/composables/useToast'

onMounted(() => {
  initToastService()
})
</script>

<template>
  <main
    class="bg-base-300 flex h-screen w-full overflow-auto bg-[url(/sasso-pattern.png)] bg-size-[3000px] bg-center bg-repeat"
  >
    <RouterView />
    <ToastContainer />
  </main>
</template>

<!-- TODO: ip/24 converter -->
