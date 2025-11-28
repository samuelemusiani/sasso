<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  title: string
  create: (event: SubmitEvent) => void
  error?: string
  hideCreate?: boolean
  open?: boolean
  disabled?: boolean
}>()

const $emit = defineEmits<{
  (e: 'close'): void
}>()

const openCreate = ref(props.open ?? false)

watch(
  () => props.open,
  (newVal) => {
    openCreate.value = newVal ?? false
  },
)

function openClose() {
  openCreate.value = !openCreate.value
  if (!openCreate.value) {
    $emit('close')
  }
}
</script>

<template>
  <div>
    <button class="btn btn-primary rounded-xl" :disabled="props.disabled" @click="openClose">
      <IconVue v-if="!openCreate" icon="mi:add" class="text-xl transition"></IconVue>
      <IconVue v-else icon="material-symbols:close-rounded" class="text-xl transition"></IconVue>
      {{ openCreate ? 'Close' : (props.hideCreate ? '' : 'Create ') + `${props.title}` }}
    </button>
  </div>
  <div v-if="openCreate">
    <form
      class="border-primary bg-base-200 flex h-full w-full flex-col gap-4 rounded-xl border p-4"
      @submit.prevent="props.create"
    >
      <slot></slot>
      <p v-if="props.error" class="text-error">{{ props.error }}</p>
      <button class="btn btn-success rounded-lg p-2" type="submit">
        {{ (props.hideCreate ? '' : 'Create ') + props.title }}
      </button>
    </form>
  </div>
</template>
