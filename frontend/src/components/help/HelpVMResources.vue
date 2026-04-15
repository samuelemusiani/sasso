<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>This page allows you to change the resources of your virtual machine.</p>
      <p>
        You can always add and remove cores and RAM, but you can only increase the disk, you cannot
        decrease it.
      </p>
      <p>
        The changes you make here will be applied at the next start of the VM. Keep in mind that a
        reboot from inside the VM will not apply the changes, you need shutdown and start the VM in
        two separate steps.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
