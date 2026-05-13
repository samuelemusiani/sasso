<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        Groups allows users to have shared VMs in order to collaborate on projects. All users in a
        group have access to the same VMs, but only admins and owners of the group can manage the
        group and its VMs.
      </p>
      <p>
        Groups are built on the idea that every member donates some of their resources to the group,
        so the more members a group has, the more resources it has. This allows to create larger VMs
        that a single user would not be able to create on their own.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Create a Group</template>
      <p>
        Creating a group only requires the name and optionally the description of the group. After
        creating the group, you will be the owner of the group and you will be able to invite other
        users to join the group and manage the group and its VMs.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Invitations</template>
      <p>
        Under the "Invitations" part of the group page, you can see the list of pending invitations
        for you made by other users to join their groups. You can accept or reject these
        invitations. If you accept an invitation, you will become a member of the group and you will
        have access to the group's VMs.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
