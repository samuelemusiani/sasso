<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>This page allows you to change the resources of your virtual machine.</p>
      <p>The resources can be changed only while the VM is stopped.</p>
      <p>
        You can add and remove cores and RAM, but you can only increase the disk, you cannot
        decrease it.
      </p>
      <p>
        The change is applied a few seconds after you request it and is in effect when you start the
        VM again.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
