<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()

function computeRoute(s: string) {
  const index = route.path.indexOf(s)
  return route.path.slice(0, index + s.length)
}
</script>

<template>
  <div class="breadcrumbs mb-2">
    <ul>
      <li v-for="value in route.path.split('/').slice(1)" :key="value">
        <RouterLink
          :to="`${computeRoute(value)}`"
          class="label hover:text-base-content font-semibold capitalize"
          >{{ value }}</RouterLink
        >
      </li>
    </ul>
  </div>
</template>
