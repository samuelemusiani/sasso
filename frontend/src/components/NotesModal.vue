<script setup lang="ts">
import { ref } from 'vue'

const $props = defineProps<{
  title: string
  body: string
}>()

const open = ref(false)

const notesModalRef = ref<HTMLDialogElement | null>(null)

function openDialog() {
  open.value = true
  const el = notesModalRef.value
  if (!el) return
  if (!el.open) el.showModal()
}

function closeDialog() {
  open.value = false
  const el = notesModalRef.value
  if (!el) return
  if (el.open) el.close()
}

const truncateLength = 50

function truncateNotes(notes: string, length: number = truncateLength) {
  if (notes.length > length) {
    return notes.substring(0, length - 3) + '...'
  }
  return notes
}
</script>

<template>
  <div>
    <div class="hover:link" @click="openDialog()">
      {{ truncateNotes($props.body) }}
    </div>

    <dialog ref="notesModalRef" class="modal modal-bottom sm:modal-middle">
      <div class="modal-box">
        <h3 class="mb-4 text-xl font-bold">{{ $props.title }}</h3>
        <p class="text-balance">{{ $props.body }}</p>

        <div class="modal-action">
          <button class="btn rounded-lg" type="button" @click="closeDialog()">Close</button>
        </div>
      </div>

      <form method="dialog" class="modal-backdrop">
        <button aria-label="Close"></button>
      </form>
    </dialog>
  </div>
</template>
