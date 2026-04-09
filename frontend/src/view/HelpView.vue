<script setup lang="ts">
import { computed, type Component, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute, type RouteRecordRaw } from 'vue-router'
import { getPageIcon } from '@/const'
import BreadcrumbNav from '@/components/BreadcrumbNav.vue'
import HelpListItem from '@/components/help/HelpListItem.vue'
import { useUiStore } from '@/stores/ui'

const router = useRouter()
const route = useRoute()
const ui = useUiStore()

const isBaseHelpRoute = computed(() => route.path === '/help' || route.path === '/help/')

type routeMeta = {
  title: string
}

type ruteRecord = {
  path: string
  component: Component
  meta: routeMeta
  children?: ruteRecord[]
}

function mapChildRoutes(routes: RouteRecordRaw[]): ruteRecord[] {
  return routes
    .slice()
    .filter((r) => r.path !== '')
    .map((r) => {
      if (r.children) {
        return { ...r, children: mapChildRoutes(r.children) }
      }
      return r
    }) as ruteRecord[]
}

const helpChildRoutes = computed(() => {
  // Find the /help route record
  const helpRoute = router.getRoutes().find((r) => r.path === '/help')
  // Its children become normalized absolute paths like "/help/vm"
  const tmp = mapChildRoutes(helpRoute?.children || [])
  console.log('help child routes', tmp)
  return tmp
})

let wasHelpOpen = false

onMounted(() => {
  wasHelpOpen = ui.helpOpen
  ui.closeHelp()
})

onBeforeUnmount(() => {
  if (wasHelpOpen) {
    ui.openHelp()
  }
})
</script>

<template>
  <div class="flex flex-col gap-2 p-2">
    <h1 class="flex items-center gap-2 text-3xl font-bold">
      <IconVue class="text-primary" :icon="getPageIcon('help')"></IconVue>Help Page
    </h1>
    <BreadcrumbNav />

    <div v-if="isBaseHelpRoute" class="p-4">
      <HelpListItem v-if="isBaseHelpRoute" base-path="/help" :routes="helpChildRoutes" />
    </div>

    <div v-else class="help-content">
      <!-- renders HelpVMs, HelpVPN, etc. -->
      <RouterView />
    </div>
  </div>
</template>
