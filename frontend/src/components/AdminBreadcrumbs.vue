<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="breadcrumbs mb-2">
    <ul>
      <li v-for="value in route.path.split('/').slice(1)" :key="value">
        <RouterLink
          :to="`/${value}`"
          class="capitalize font-semibold label hover:text-base-content"
          >{{ value }}</RouterLink
        >
      </li>
    </ul>
  </div>
</template>
