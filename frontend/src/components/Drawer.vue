<script setup lang="ts">
import DrawerLine from "@/components/DrawerLine.vue";

const menu = {
  Home: { icon: "material-symbols--home-rounded", to: "/" },
  VM: { icon: "mi--computer", to: "/vm" },
  Nets: { icon: "ph--network", to: "/net" },
  "SSH Keys": { icon: "icon-park-twotone--key", to: "/ssh-keys" },
  VPN: { icon: "arcticons--wireguard", to: "/vpn" },
};
</script>

<template>
  <div class="drawer" :class="{ 'drawer-open': drawer }">
    <input id="my-drawer" type="checkbox" class="drawer-toggle" />
    <div class="drawer-content">
      <!-- Page content here -->
      <label
        for="my-drawer"
        class="btn btn-primary drawer-button"
        @click="drawer = !drawer"
      >
        <span class="icon-[material-symbols--menu-rounded]"></span>
      </label>
    </div>
    <div class="drawer-side">
      <label for="my-drawer" aria-label="close sidebar" class="drawer-overlay"></label>
      <ul
        class="menu bg-base-100/70 rounded-lg min-h-full w-64 p-4 *:active:bg-base-content/10 *:active:text-base-100"
      >
        <!-- Sidebar content here -->
        <li v-for="(item, name) in menu" :key="name">
          <DrawerLine :to="item.to" :icon="item.icon" :label="name"></DrawerLine>
        </li>

        <RouterLink to='/settings' class="rounded-lg min-w-32 block text-left">
          Settings
        </RouterLink>
        <button @click="logout" class="rounded-lg min-w-32 block text-left">
          Logout
        </button>
      </ul>
    </div>
  </div>
</template>
