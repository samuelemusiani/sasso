<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'

type Decision = 'positive' | 'negative'

const props = withDefaults(
  defineProps<{
    /** Controls visibility from the parent (v-model) */
    modelValue: boolean

    title?: string

    positiveText: string
    negativeText: string

    /** Optional: disable closing via ESC/backdrop */
    disableCancel?: boolean
  }>(),
  {
    title: 'Confirm',
    disableCancel: false,
  },
)

const emit = defineEmits<{
  /** v-model sync */
  (e: 'update:modelValue', value: boolean): void

  /** final decision */
  (e: 'decision', value: Decision): void

  /** convenience events (optional to use) */
  (e: 'positive'): void
  (e: 'negative'): void
}>()

const dialogRef = ref<HTMLDialogElement | null>(null)

function openDialog() {
  const el = dialogRef.value
  if (!el) return
  if (!el.open) el.showModal()
}

function closeDialog() {
  const el = dialogRef.value
  if (!el) return
  if (el.open) el.close()
}

/**
 * Close initiated by our buttons
 */
function choose(decision: Decision) {
  emit('decision', decision)
  if (decision === 'positive') emit('positive')
  else emit('negative')

  emit('update:modelValue', false)
  closeDialog()
}

/**
 * ESC/backdrop close => negative
 * - dialog 'cancel' event fires on ESC (and can be prevented)
 * - dialog 'close' event fires on any close (including el.close())
 */
function onCancel(e: Event) {
  if (props.disableCancel) {
    e.preventDefault()
    return
  }
  // ESC means negative decision
  choose('negative')
}

function onClose() {
  // If parent still thinks it's open, sync it closed.
  // (e.g., user clicked backdrop; dialog closed itself)
  if (props.modelValue) emit('update:modelValue', false)
}

watch(
  () => props.modelValue,
  (isOpen) => {
    if (isOpen) openDialog()
    else closeDialog()
  },
)

onMounted(() => {
  const el = dialogRef.value
  if (!el) return

  el.addEventListener('cancel', onCancel)
  el.addEventListener('close', onClose)

  // If parent starts with true
  if (props.modelValue) openDialog()
})

onBeforeUnmount(() => {
  const el = dialogRef.value
  if (!el) return
  el.removeEventListener('cancel', onCancel)
  el.removeEventListener('close', onClose)
})
</script>

<template>
  <dialog ref="dialogRef" class="modal modal-bottom sm:modal-middle">
    <div class="modal-box">
      <h3 class="mb-4 text-xl font-bold">{{ title }}</h3>
      <slot />

      <div class="modal-action flex justify-between">
        <!-- Negative -->
        <button class="btn" type="button" @click="choose('negative')">
          {{ negativeText }}
        </button>

        <!-- Positive -->
        <button class="btn btn-primary" type="button" @click="choose('positive')">
          {{ positiveText }}
        </button>
      </div>
    </div>

    <!-- Optional backdrop click closes the dialog (counts as negative via close-sync + cancel/close behavior) -->
    <form v-if="!disableCancel" method="dialog" class="modal-backdrop">
      <button aria-label="Close"></button>
    </form>
  </dialog>
</template>
