<template>
  <div class="p-2">
    <div>Admin view for <b>sasso</b>!</div>
    <div class="flex gap-2">
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
        class="bg-green-400 hover:bg-green-300 p-2 rounded-lg min-w-32 block text-center"
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
