<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Introduction</h2>
      <p>
        To access a VM via SSH, you need to have at least one SSH key added to your account. This
        page allows you to manage your SSH keys.
      </p>
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Generation</h2>
      <p>
        If you don't have an SSH key, you can generate one using the following command on your local
        machine:
      </p>
      <pre class="bg-base-100 overflow-x-auto rounded-lg p-4 font-mono">ssh-keygen -t ed25519 </pre>
      <p>
        This will generate a new SSH key pair (a private key and a public key). The public key is
        the one you need to add to your Sasso account, while the private key should be kept secret
        and stored securely on your local machine.
      </p>
      <p>
        During the generation process, you will be asked to enter a file name to save the key
        (usually <span class="bg-base-100 rounded-lg p-1 font-mono">~/.ssh/id_ed25519</span>). This
        is the public key and what you have to add to Sasso. An example of a public key looks like
        this:
      </p>
      <pre class="bg-base-100 overflow-x-auto rounded-lg p-4 font-mono">
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMtcEPAeo8hwYbZrhzIlF1pqp2ILSBbxo2oDoXd401+A user@pc</pre
      >
    </div>

    <div class="flex flex-col gap-2">
      <h2 class="text-info font-bold">Name</h2>
      <p>
        The "<i>Name</i>" field is just a label for you to identify the key, it has no effect on the
        actual SSH key.
      </p>
    </div>

    <div>
      <h2 class="text-info font-bold">Distribution to VMs</h2>
      <p>
        After adding or deleting an SSH key, the changes will be automatically applied to all your
        VMs. <b>ONLY STOPPED VMs</b> will be affected by the changes immediately. If you have a
        running VM, it will not receive the new SSH keys and will still have access to the deleted
        SSH keys until you restart it.
      </p>
      <p>
        After modifying your SSH keys, already accesed VMs will probably display an error like this:
      </p>
      <pre class="bg-base-100 overflow-x-auto rounded-lg p-4 font-mono">
WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED! </pre
      >
      <p>
        This is caused by the fact that every time you modify your SSH keys, the SSH server on your
        VMs regenerates its host keys. If you are using the VPN you are safe. Just follow the error
        message on how to update the host key on your local machine.
      </p>
    </div>
  </div>
</template>
