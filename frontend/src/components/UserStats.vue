<script setup lang="ts">
let props = defineProps<{
	item: string
	icon: string
	percent?: number
	color: string
}>()

const colorClasses: Record<string, { bg: string; text: string }> = {
  primary: { bg: 'bg-primary/20', text: 'text-primary' },
  secondary: { bg: 'bg-secondary/20', text: 'text-secondary' },
  accent: { bg: 'bg-accent/20', text: 'text-accent' },
  info: { bg: 'bg-info/20', text: 'text-info' },
  success: { bg: 'bg-success/20', text: 'text-success' },
  warning: { bg: 'bg-warning/20', text: 'text-warning' },
  error: { bg: 'bg-error/20', text: 'text-error' },
}
</script>
<template>
	<div class="flex justify-between items-center p-4 gap-2 rounded-xl shadow-md w-full"
		:class="`bg-base-100 ${colorClasses[props.color].text}`">
		<div class="flex items-center flex-col gap-2">
			<IconVue class="text-4xl" :icon="props.icon" />
			<span class="font-semibold">{{ props.item }}</span>
		</div>
		<div class="radial-progress" :class="`text-${props.color}`" :style="`--value:${props.percent};`"
			:aria-valuenow="props.percent" role="progressbar">
			{{ props.percent }}%</div>
	</div>
</template>