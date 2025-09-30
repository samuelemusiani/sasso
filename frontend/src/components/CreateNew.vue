<script setup lang="ts">
import { ref } from 'vue'
let openCreate = ref(false)

const props = defineProps<{
	title: string
	create: (event: MouseEvent) => void
	error?: string
}>()
</script>

<template>
	<div>
		<button class="btn btn-primary rounded-xl" @click="openCreate = !openCreate">
			<IconVue v-if="!openCreate" icon="mi:add" class="text-xl transition"></IconVue>
			<IconVue v-else icon="material-symbols:close-rounded" class="text-xl transition"></IconVue>
			{{ openCreate ? 'Close' : `Create ${props.title}` }}
		</button>
	</div>
	<div v-if="openCreate">
		<div class="p-4 border border-primary rounded-xl bg-base-200 flex flex-col gap-4 w-full h-full">
			<slot></slot>
			<p v-if="props.error" class="text-error">{{ props.error }}</p>
			<button class="btn btn-success p-2 rounded-lg" @click="props.create">Create {{ props.title }}</button>
		</div>
	</div>
</template>