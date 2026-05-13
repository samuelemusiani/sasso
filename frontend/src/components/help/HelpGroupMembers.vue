<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>This page allows you to invite and remove members to and from your group.</p>
    </HelpParagraph>
    <HelpParagraph>
      <template #title>Create invitation</template>
      <p>
        To create an invitation, you need to enter the username of the user you want to invite and
        its role in the group.
      </p>
      <p>
        Admins have more powers than members, but every member will be able to access the group VMs.
      </p>
      <p>
        The username you enter must be the exact username of the user you want to invite. If the
        user is in another realm, you can specify it appending
        <CodeSnippet :inline="true">@realm</CodeSnippet> to the username, for example
        "john@external" to invite the user "john" from the "external" realm. If no realm is
        specified, the user will be searched in the same realm of who is creating the invitation.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
