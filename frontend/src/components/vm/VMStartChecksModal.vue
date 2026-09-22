<script setup lang="ts">
import ModalAlert from '@/components/ModalAlert.vue'

defineProps<{
  modelValue: boolean
  missing: { sshKeys: boolean; interfaces: boolean }
  interfacesHref: string
}>()
defineEmits(['confirm', 'cancel'])
</script>

<template>
  <ModalAlert
    :model-value="modelValue"
    title="Proceed with start?"
    positive-text="Yes, start the VM"
    negative-text="No, don't start the VM"
    @positive="$emit('confirm')"
    @negative="$emit('cancel')"
  >
    <div class="flex flex-col gap-4">
      <p v-if="missing.sshKeys">
        You have no <a class="link link-info" href="/ssh-keys" target="_blank">SSH Keys</a> in your
        account. You won't be able to connect via SSH if you start it.
      </p>

      <p v-if="missing.interfaces">
        This VM has no
        <a class="link link-info" :href="interfacesHref" target="_blank">network interfaces</a>. You
        won't be able to connect to it if you start it.
      </p>

      <p>Are you sure you want to start the VM?</p>
    </div>
  </ModalAlert>
</template>
