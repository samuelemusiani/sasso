<script setup lang="ts">
import { useRoute } from 'vue-router'
import HelpHome from '@/components/help/HelpHome.vue'
import HelpVMBackups from '@/components/help/HelpVMBackups.vue'
const route = useRoute()
import type { Component } from 'vue'

import { useUiStore } from '@/stores/ui'
const ui = useUiStore()

type selector = {
  regex: string
  component: Component
}

const helpSelectors: selector[] = [
  { regex: '^/$', component: HelpHome },
  { regex: '^/vm/[0-9]*/backups$', component: HelpVMBackups },
]

function componentFromRoute(path: string) {
  for (const selector of helpSelectors) {
    const regex = new RegExp(selector.regex)
    if (regex.test(path)) {
      return selector.component
    }
  }
  return null
}
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
            <component v-if="componentFromRoute(route.path)" :is="componentFromRoute(route.path)" />
            <div v-else class="">
              <span class="text-base-content/60">No help content for this page yet :(</span>
            </div>
          </div>
        </div>

        <button class="btn btn-outline btn-info rounded-lg" @click="ui.closeHelp()">Close</button>
      </div>
    </div>
  </div>
</template>
