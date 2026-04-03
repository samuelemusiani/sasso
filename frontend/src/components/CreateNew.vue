<script setup lang="ts">
import { ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    title: string
    // Returns true if the creation was successful, false otherwise. Can be async.
    create: (event: SubmitEvent) => Promise<boolean> | boolean | void
    error?: string
    hideCreate?: boolean
    open?: boolean
    disabled?: boolean
    loading?: boolean
    closeOnCreate?: boolean
  }>(),
  {
    closeOnCreate: false,
  },
)

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

async function submit(event: SubmitEvent) {
  const r = await props.create(event)
  if (r === undefined) {
    return
  }

  if (props.closeOnCreate && r) {
    openCreate.value = false
    $emit('close')
  }
}

function openClose() {
  openCreate.value = !openCreate.value
  if (!openCreate.value) {
    $emit('close')
  }
}
</script>

<template>
  <div class="flex w-full flex-col gap-2">
    <div>
      <button class="btn btn-primary rounded-xl" :disabled="props.disabled" @click="openClose">
        <IconVue v-if="!openCreate" icon="mi:add" class="text-xl transition"></IconVue>
        <IconVue v-else icon="material-symbols:close-rounded" class="text-xl transition"></IconVue>
        {{ openCreate ? 'Close' : (props.hideCreate ? '' : 'Create ') + `${props.title}` }}
      </button>
    </div>
    <div v-if="openCreate" class="w-full">
      <form
        class="border-primary bg-base-200 flex h-full w-full flex-col gap-4 rounded-xl border p-4"
        @submit.prevent="submit"
      >
        <slot></slot>
        <p v-if="props.error" class="text-error">{{ props.error }}</p>
        <button class="btn btn-success rounded-lg p-2" type="submit" :disabled="loading">
          <div v-if="loading" class="grid h-70">
            <span class="loading loading-spinner place-self-center"></span>
          </div>
          {{ (props.hideCreate ? '' : 'Create ') + props.title }}
        </button>
      </form>
    </div>
  </div>
</template>
