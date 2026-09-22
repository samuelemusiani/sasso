<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>This page allows you to donate and take back resources to and from your group.</p>
      <p>The page shows the aggregate resources of the group.</p>
      <p>After a group has more than 2 users, a network is automatically allocated to the group.</p>
    </HelpParagraph>
  </HelpPage>
</template>
