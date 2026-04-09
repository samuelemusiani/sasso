<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        Welcome to the Help Panel of Sasso! Here you can find explanations and small guides that
        will help you navigate through the different features of Sasso.
      </p>
      <p>
        This page can always be accessed by clicking the "HELP" button on the top right corner of
        the screen.
      </p>
      <p>
        The content of this page will change depending on which page you are currently on, so make
        sure to check it out whenever you need help with something!
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Resources</template>
      <p>
        On this page you can find a summary of every resource available to you! CPU, RAM, Disk and
        Networks.
      </p>
      <p>
        The resources are divided into 4 categories: <i>Active</i>, <i>Allocated</i> and
        <i>Group shared</i>.
      </p>
      <p>
        <b class="text-info">Active</b> resources are the ones currently being used by your
        <i>running</i> VMs. Networks are a special case, as they are active as soon as they are
        created, regardless of the actual usage of them.
      </p>
      <p>
        <b class="text-info">Allocated</b> resources are the ones reserved for your VMs, but not
        necessarily being used at the moment. For example, if you have a VM that is currently
        stopped, its resources will be allocated but not active.
      </p>
      <p>
        <b class="text-info">Group shared</b> resources are the ones that you have allocated to
        groups. Learn more about groups in the
        <router-link to="/group" class="link link-info"> Groups section </router-link>
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
