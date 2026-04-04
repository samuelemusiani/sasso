<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Introduction</h2>
      <p>
        Sasso uses a VPN service based on
        <a class="link link-error" href="https://www.wireguard.com/" target="_blank">WireGuard</a>.
        This allows you to connect to your VMs as if they were directly connected to your local
        network, providing a secure and fast connection to them.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Security</h2>
      <p>
        All the traffic between your device and your VMs that goes through the VPN is
        <b>encrypted</b> using WireGuard's encryption protocols. This ensures that no one exept you
        and the Sasso servers can access the data being transmitted, providing a secure connection
        even over public insecure networks.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Configurations</h2>
      <p>
        Wireguard uses a profile (configuration) to define a connection. Refer to the offical
        documentation on how to use them. If you have the
        <span class="bg-base-100 rounded-lg p-1 font-mono">wg-quick</span> command line tool
        installed, you can use the following command to activate a profile:
      </p>
      <pre class="bg-base-100 overflow-x-auto rounded-lg p-4 font-mono">
 wg-quick up /path/to/profile.conf </pre
      >
    </div>
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Multiple profiles</h2>
      <p>
        In this page you can find multiple profiles (configurations) that you can use to connect to
        your VMs. Every profile is equivalent and they all have the permission to connect to all
        your VMs. Multiple profiles are provided so you can use different devices at the same time.
        For example, when developing a website on a VM, you can use one profile for you laptop and
        another one for your phone, so you can test the website on both devices at the same time.
      </p>
      <p>
        You can create a new profile by clicking the "<i>Create New VPN Profile</i>" button at the
        bottom. This will generate a new unique profile. Keep in mind that you have a limited number
        of profiles :)
      </p>
      <p>
        If you want to delete a profile or regenerate it (which will invalidate the previous one),
        just click "<i>Delete configuration</i>" button on the top right of each profile.
      </p>
    </div>
  </div>
</template>
