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
      class="border-primary bg-base-200 flex h-full w-full flex-col gap-4 rounded-xl border p-4"
      @submit.prevent="props.create"
    >
      <slot></slot>
      <p v-if="props.error" class="text-error">{{ props.error }}</p>
      <button class="btn btn-success rounded-lg p-2" type="submit">Create {{ props.title }}</button>
    </form>
  </div>
</template>
