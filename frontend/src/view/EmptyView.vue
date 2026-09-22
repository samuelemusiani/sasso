<!--
This is simple view that just renders the child route.
It is used for routes like /vm and /vm/:vmid that are based on different views.
It can be used freely.
-->

<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <RouterView />
</template>
