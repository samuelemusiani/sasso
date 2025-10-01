<script setup lang="ts">
import { ref } from 'vue'
const openCreate = ref(false)

const props = defineProps<{
  title: string
  create: (event: SubmitEvent) => void
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
    <form
      class="p-4 border border-primary rounded-xl bg-base-200 flex flex-col gap-4 w-full h-full"
      @submit.prevent="props.create"
    >
      <slot></slot>
      <p v-if="props.error" class="text-error">{{ props.error }}</p>
      <button class="btn btn-success p-2 rounded-lg" type="submit">Create {{ props.title }}</button>
    </form>
  </div>
</template>
