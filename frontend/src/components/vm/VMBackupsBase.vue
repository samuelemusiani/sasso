<script setup lang="ts">
import { useRoute } from 'vue-router'
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { VM } from '@/types'

const $props = defineProps<{
  vm: VM
}>()

const route = useRoute()

const backupList = computed(() => {
  return route.path.endsWith('/backups')
})
</script>

<template>
  <div>
    <RouterLink
      :to="backupList ? 'backups/requests' : route.path.substring(0, route.path.length - 9)"
      class="btn btn-outline btn-secondary absolute top-16 right-2 w-52 rounded-lg"
    >
      <IconVue
        :icon="backupList ? 'material-symbols:history' : 'material-symbols:backup-outline'"
        class="text-lg"
      />
      {{ backupList ? 'Requests History' : 'Backups' }}
    </RouterLink>

    <router-view :vm="$props.vm" />
  </div>
</template>
