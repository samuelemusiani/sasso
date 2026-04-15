<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>In this page you find the general information about the virtual machine.</p>
      <p>You can also start, stop, restart or delete the virtual machine.</p>
      <p>
        All VMs have a lifetime and you can extend it here. Be aware that you can only extend the
        lifetime of the VM if you have enough resources in your account.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
