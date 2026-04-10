<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        This page is a simple list of all interfaces that are present on your VMs. It's useful when
        the interface creation form reports that an IP is already in use, but you don't know which
        VM is using it. Or it's also useful to have a global view of all your interfaces and all the
        attached information about them, such as the IP, gateway, VM they belongs to, the network
        it's connected to and so on.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Fields</template>
      <p>
        The fields are pretty self explanatory, but if you want to know more about them, go read the
        help page inside the VM details, in the "Interfaces" section. It has a more detailed
        explanation of all the fields and all the possible states of an interface.
      </p>
      <p>
        You can directly access the "Interfaces" section of the VM details by clicking the name of
        the VM under the "VM Name" column.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
