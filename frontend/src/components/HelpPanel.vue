<script setup lang="ts">
import { useRoute, RouterLink } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()
const helpComponent = computed(() => route.meta.helpComponent)

import { useUiStore } from '@/stores/ui'
const ui = useUiStore()

const fullHelpRoute = computed(() => {
  return '/help' + route.path
})
</script>

<template>
  <div v-if="ui.helpOpen" class="flex max-h-screen overflow-hidden">
    <div
      class="from-base-200 to-base-200/40 my-4 flex w-80 flex-col items-center rounded-xl bg-linear-to-r shadow-lg backdrop-blur-md transition-all duration-300"
    >
      <div class="flex h-full w-full flex-col justify-between p-4">
        <!-- Toggle -->
        <div class="flex w-full flex-col gap-4">
          <div class="flex w-full items-center justify-center gap-2">
            <h1 class="text-info text-xl font-bold">Help Page</h1>
          </div>

          <div class="h-[82vh] overflow-y-auto pr-2">
            <component v-if="helpComponent" :is="helpComponent" />
            <div v-else class="">
              <span class="text-base-content/60">No help content for this page yet :(</span>
            </div>
          </div>
        </div>

        <div class="flex justify-between gap-2">
          <button class="btn btn-outline flex-1 rounded-lg" @click="ui.closeHelp()">
            <IconVue icon="material-symbols:close" class="text-lg" />
            Close
          </button>
          <RouterLink
            :to="fullHelpRoute"
            class="btn btn-info flex-1 rounded-lg"
            @click="ui.closeHelp()"
          >
            <IconVue icon="iconoir:page" class="text-lg" />
            Full Page
          </RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>
