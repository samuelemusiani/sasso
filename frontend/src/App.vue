<script setup lang="ts">
import { RouterView, useRoute } from 'vue-router'
import { computed } from 'vue'
import Drawer from '@/components/AppDrawer.vue'
import NotificationContainer from '@/components/NotificationContainer.vue'

const route = useRoute()

// Nascondi il drawer per la pagina di login
const showDrawer = computed(() => route.path !== '/login')
</script>

<template>
  <main
    v-if="showDrawer"
    class="flex w-full h-screen bg-[url(/pattern-sasso.png)] bg-base-300 bg-center bg-repeat bg-size-[3000px]"
  >
    <Drawer />
    <div class="flex-1 overflow-auto">
      <div
        class="backdrop-blur-xl p-4 rounded-2xl m-6 bg-gradient-to-br from-primary/10 via-secondary/5 to-accent/10 p-6"
      >
        <RouterView />
      </div>
    </div>
  </main>

  <!-- Layout per pagina di login senza drawer -->
  <main v-else class="w-full h-screen">
    <RouterView />
  </main>

  <!-- Container delle notifiche (sempre presente) -->
  <NotificationContainer />
</template>
