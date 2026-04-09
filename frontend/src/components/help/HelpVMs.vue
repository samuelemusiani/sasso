<script setup lang="ts">
import { getStatusClass } from '@/const'
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Introduction</h2>
      <p>
        VMs are the core of sasso. In this page you can find a summary of all yours VMs - even the
        group shared ones - and some relevant information about them. More detailed information
        about each VM can be found in the VM details page, which you can access by clicking the
        "Manage" button on the right of each VM.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Fields</h2>
      <p>
        <b class="text-info">Name</b> is the name and <b>hostname</b> of your VM. Use it to
        distinguish between your VMs.
      </p>
      <p>
        <b class="text-info">Notes</b> is a free text field that you can use to write any notes
        about the VM. It only helps you. It's usually a good idea to write the OS image the VM is
        based on, beucase it changes the way to access the VM.
      </p>
      <p>
        <b class="text-info">Group</b>: if "Me" the VM belongs to you, otherwise it belongs to the
        group specified.
      </p>
      <p>
        <b class="text-info">CPU, RAM, Disk</b> are the resources allocated to the VM. If it's your
        VM they are taken from your personal resources, otherwise they are taken from the group
        shared resources. The minimum is 1 CPU, 512 MB of RAM and the disk size depends on the OS
        image you choose when creating the VM.
      </p>
      <p>
        <b class="text-info">Status</b>: VMs can have a lot of different states. If a state is
        prefixed with "<i>Pre-</i>" it means that the VM is waiting to enter the state without the
        prefix. For example, "Pre-Creating" means that the VM is waiting to be actually created and
        enter the "Creating" state.
      </p>
      <ul class="list-disc pl-6">
        <li>
          <b :class="getStatusClass('pre-creating')">Pre-Creating</b>: the VM is waiting to be
          created.
        </li>
        <li>
          <b :class="getStatusClass('creating')">Creating</b>: the VM is being created. This process
          can take a while, be patient!
        </li>
        <li>
          <b :class="getStatusClass('pre-configuring')">Pre-Configuring</b>: the VM is waiting to be
          configured.
        </li>
        <li>
          <b :class="getStatusClass('configuring')">Configuring</b>: the VM is being configured.
          This process only happens for some combinations of OS images and resources.
        </li>
        <li>
          <b :class="getStatusClass('stopped')">Stopped</b>: the VM is created but not running. You
          can start it by clicking the "Start" button on the right of it.
        </li>
        <li>
          <b :class="getStatusClass('running')">Running</b>: the VM is running and you can connect
          to it.
        </li>
        <li>
          <b :class="getStatusClass('pre-deleting')">Pre-Deleting</b>: the VM is waiting to be
          deleted.
        </li>
        <li><b :class="getStatusClass('deleting')">Deleting</b>: the VM is being deleted.</li>
        <li>
          <b :class="getStatusClass('unknown')">Unknown</b>: the VM is in an unknown state. Contact
          the administrator if you see this.
        </li>
      </ul>
      <p>
        <b class="text-info">Lifetime</b> VMs do not have infinite lifetime by their own. A lifetime
        is the date the VM will expire and automatically deleted. This helps to free resources that
        are not being used anymore. You can always extend the lifetime of your VMs by going into the
        VM details page.
      </p>
      <p>
        <b class="text-info">ID</b> it's a unique numeric identifier for the VM. It's useful for
        debugging and to identify the VM when contacting the administrator. You can find it in the
        VM details page or using the "Show IDs" toggle on the top right of the page.
      </p>
    </div>
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Actions</h2>
      <p>You can perform various actions on your VMs:</p>
      <ul class="list-disc pl-6">
        <li>
          <b class="text-success">Start</b>: starts the VM. Only possible if the VM is in "Stopped"
          state. It can take a while because the VM needs to boot, be patient!
        </li>
        <li>
          <b class="text-warning">Stop</b>: stops a running VM. This is like pulling the power cord,
          so it <b><u>can cause data loss</u></b> if the VM is not properly shut down. Use it only
          if the VM is not responding and you cannot stop it gracefully from the VM details page.
        </li>
        <li>
          <b class="text-info">Restart</b>: restarts a running VM. This is like pulling the power
          cord and plugging it back in, so it <b><u>can cause data loss</u></b> if the VM is not
          properly shut down. Use it only if the VM is not responding and you cannot restart it
          gracefully from the VM details page.
        </li>
        <li>
          <b class="text-error">Delete</b>: deletes a VM. This action is irreversible and you will
          lose all the data on the VM.
        </li>
      </ul>
    </div>
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Access</h2>
      <p>
        To access the VM you first need to add an
        <RouterLink to="/ssh-keys" class="link link-primary">SSH key to your account</RouterLink>,
        active the <RouterLink to="/vpn" class="link link-primary">VPN</RouterLink> on your
        computer, add an <b>interface</b> to the VM and then connect to it using SSH and the IP
        address of the interface.
      </p>
      <p>
        The <b class="text-info">User</b> to use to connect to the VM depends on the OS image the VM
        is based on. For example, if the VM is based on an Ubuntu image, the user is "ubuntu", i f
        the VM is based on Debian, the user is "debian" and so on.
      </p>
      <p>The command to connect to the VM is usually something like this:</p>
      <pre class="bg-base-100 overflow-x-auto rounded-lg py-4 font-mono">
        ssh &lt;user&gt;@&lt;ip&gt; </pre
      >
      <p>
        For example, if the OS image is Debian and the IP address of the interface is
        <span class="bg-base-100 rounded-lg p-1 font-mono">10.0.0.1</span>, the command would be:
      </p>
      <pre class="bg-base-100 overflow-x-auto rounded-lg py-4 font-mono">
        ssh debian@10.0.0.1 </pre
      >
      <p>
        Please note that <b class="text-info">SSH Keys</b> only apply to "Stopped" VMs. If you add
        an SSH key to a VM that is already running, the key will not be added until the next boot.
      </p>
    </div>
  </div>
</template>
