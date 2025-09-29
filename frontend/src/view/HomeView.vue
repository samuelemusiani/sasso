<script setup lang="ts">
import { onMounted, ref } from 'vue';
import UserStats from '@/components/UserStats.vue';
import { api } from '@/lib/api';
import type { User } from '@/types';

const whoami = ref<User | null>(null);
let stats = ref();

function fetchWhoami() {
  api
    .get("/whoami")
    .then((res) => {
      whoami.value = res.data as User;
      stats.value = [
        { item: "Core", icon: "heroicons-solid:chip", percent: whoami.value.max_cores , color: 'primary'},
        { item: "Nets", icon: "ph:network", percent: whoami.value.max_nets, color: 'secondary'},
        { item: "RAM", icon: "fluent:ram-20-regular", percent: whoami.value.max_ram, color: 'success' },
        { item: "Disk", icon: "mingcute:storage-line", percent: whoami.value.max_disk, color: 'accent' },
      ]
    })
    .catch((err) => {
      console.error("Failed to fetch whoami:", err);
    });

}

onMounted(() => {
  fetchWhoami();
});

</script>

<template>
  <div class="h-full w-full">
    <h1 class="text-3xl font-bold my-3">Hi {{ whoami?.username }}!</h1>
      <h1 class="text-xl font-semibold my-2">Usage of your resources</h1>
    <div class="flex justify-between gap-4">
      <UserStats v-for="stat in stats" :key="stat.item" :item="stat.item" :icon="stat.icon" :percent="stat.percent" :color="stat.color" />
    </div>
  </div>
</template>
