<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import type { User } from "@/types";
import { api } from "@/lib/api";
import { useRouter } from "vue-router";

const whoami = ref<User | null>(null);
const router = useRouter();

function fetchWhoami() {
  api
    .get("/whoami")
    .then((res) => {
      console.log("Whoami response:", res.data);
      whoami.value = res.data as User;
    })
    .catch((err) => {
      console.error("Failed to fetch whoami:", err);
    });
}

const showAdminPanel = computed(() => {
  if (!whoami.value) return false;
  return whoami.value.role === "admin";
});

function logout() {
  localStorage.removeItem("jwt_token");
  router.push("/login");
}

onMounted(() => {
  fetchWhoami();
});

let drawer = ref(true);
</script>

<template>
  <div class="bg-linear-to-r to-primary/50 from-base-100">
    <div>Home view for <b>sasso</b>!</div>
    <div v-if="whoami">
      {{ whoami }}
    </div>

    <div class="flex gap-2"></div>
  </div>
</template>
