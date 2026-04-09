<script setup lang="ts">
import { getPageIcon } from '@/const'
import HelpListItem from '@/components/help/HelpListItem.vue'
import { RouterLink } from 'vue-router'
import type { Component } from 'vue'

type routeMeta = {
  title: string
}

type routeRecord = {
  path: string
  component: Component
  meta: routeMeta
  children?: routeRecord[]
}

const $props = defineProps<{
  routes: routeRecord[]
  basePath?: string
}>()
</script>

<template>
  <div v-if="$props.routes.length > 0">
    <ul class="flex flex-col gap-2">
      <li class="" v-for="r in $props.routes" :key="r.path">
        <div class="border-b-base-100 border-b hover:rounded-lg">
          <RouterLink
            class="hover:bg-base-100 text-primary flex rounded-lg p-2 capitalize"
            :to="`${$props.basePath}/${r.path}`"
          >
            <!-- show something nicer if you set meta.title -->
            <div class="flex items-center gap-2" v-if="r.meta">
              <IconVue :icon="getPageIcon(r.meta.title)" class="text-primary inline text-2xl" />
              <div v-if="r.meta" class="font-semibold">
                {{ r.meta.title }}
              </div>
            </div>
          </RouterLink>
        </div>
        <div v-if="r.children" class="ml-8 pt-2">
          <HelpListItem :base-path="`${$props.basePath}/${r.path}`" :routes="r.children" />
        </div>
      </li>
    </ul>
  </div>
</template>
