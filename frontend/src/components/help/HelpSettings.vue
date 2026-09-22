<script setup lang="ts">
import { RouterLink } from 'vue-router'
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        This page allows you to chose which type of notifications you want to receive and how you
        want to receive them. For each type you can choose to receive with email and/or with a
        <RouterLink to="/telegram" class="link link-primary">telegram bot</RouterLink> or just not
        receive them at all. Just toggle the option and it will be automatically saved.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
