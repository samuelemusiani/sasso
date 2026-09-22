<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>In this page you find the general information about the group.</p>
      <p>
        You can modify the title and the description of the group, but only if you are the owner.
      </p>
      <p>
        If you are the owner you can also delete the group, but be aware that this action is
        irreversible. This action can be performed only if the group has no VMs and no Nets, so you
        have to delete all the VMs and Nets before deleting the group.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
