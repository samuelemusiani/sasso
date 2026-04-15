<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        In this page there is a list all of the backup requests of the virtual machine. You can also
        see pending and failed backup requests.
      </p>
    </HelpParagraph>
    <HelpParagraph>
      <template #title>Fields</template>
      <p>
        <b class="text-info">ID</b> it's unique for each backup request and it's used to identify
        it.
      </p>
      <p>
        <b class="text-info">Created At</b> is the date and time when the backup request was made.
      </p>
      <p>
        <b class="text-info">Backup ID</b> is set only for
        <span class="text-error">delete</span> backup requests and it's the ID of the backup deleted
        by the request.
      </p>
      <p>
        <b class="text-info">Type</b> is the type of backup request. It can be either
        <span class="text-success">create</span>, <span class="text-error">delete</span> or
        <span class="text-warning">restore</span>.
      </p>
      <p>
        <b class="text-info">Status</b> is the status of the backup request. It can be either
        <span class="text-warning">pending</span>, <span class="text-error">failed</span> or
        <span class="text-warning">completed</span>.
      </p>
      <p>
        <b class="text-info">Name</b> is the name of the backup created by the backup request. It's
        only set for <span class="text-success">create</span> backup requests.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
