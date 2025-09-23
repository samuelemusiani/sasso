<script setup lang="ts">
import { RouterLink } from 'vue-router'

const $props = defineProps<{
  to: string
  icon: string
  label: string
}>()

</script>
<template>
	
          <RouterLink
            class="rounded-5xl min-w-32 flex items-center text-left hover:bg-primary/20 transition-colors gap-2"
            active-class="backdrop-blur-sm bg-white/80 font-semibold text-base-100"
			:to="$props.to"
          >
            <div class="btn btn-square btn-sm rounded-xl btn-primary p-0 m-1">
  <span :class="`icon-[${$props.icon}] text-lg`"></span>
            </div>
            {{$props.label}}
          </RouterLink>
</template>