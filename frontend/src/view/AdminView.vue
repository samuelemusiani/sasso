<template>
  <div class="p-2">
    <!-- Header -->
    <div class="mb-8 px-2">
      <div class="flex items-center gap-3 mb-4 text-3xl font-bold">
        <IconVue icon="material-symbols:admin-panel-settings" class="text-primary" />
        <h1 class="text-base-content">Admin Panel</h1>
      </div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8 px-2">
      <RouterLink
        class="bg-blue-400 hover:bg-blue-300 p-2 rounded-lg min-w-32 block text-center"
        to="/admin/users"
      >
        Users
      </RouterLink>
      <RouterLink
        class="bg-pink-400 hover:bg-pink-300 p-2 rounded-lg min-w-32 block text-center"
        to="/admin/realms"
      >
        Realms
      </RouterLink>
      <RouterLink
        class="bg-green-400 hover:bg-success p-2 rounded-lg min-w-32 block text-center"
        to="/admin/ssh-keys"
      >
        SSH Keys
      </RouterLink>
      <RouterLink
        class="bg-orange-400 hover:bg-orange-300 p-2 rounded-lg min-w-32 block text-center"
        to="/admin/port-forwards"
      >
        Port Forwards
      </RouterLink>
    </div>
  </div>
</template>
