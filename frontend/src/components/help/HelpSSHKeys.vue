<script setup lang="ts">
import CodeSnippet from '@/components/help/CodeSnippet.vue'
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        To access a VM via SSH, you need to have at least one SSH key added to your account. This
        page allows you to manage your SSH keys.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Generation</template>
      <p>
        If you don't have an SSH key, you can generate one using the following command on your local
        machine:
      </p>
      <CodeSnippet>ssh-keygen -t ed25519 </CodeSnippet>
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
      <CodeSnippet>
        ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMtcEPAeo8hwYbZrhzIlF1pqp2ILSBbxo2oDoXd401+A user@pc
      </CodeSnippet>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Name</template>
      <p>
        The "<i>Name</i>" field is just a label for you to identify the key, it has no effect on the
        actual SSH key.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title> Distribution to VMs </template>
      <p>
        After adding or deleting an SSH key, the changes will be automatically applied to all your
        VMs. <b>ONLY STOPPED VMs</b> will be affected by the changes immediately. If you have a
        running VM, it will not receive the new SSH keys and will still have access to the deleted
        SSH keys until you restart it.
      </p>
      <p>
        After modifying your SSH keys, already accesed VMs will probably display an error like this:
      </p>
      <CodeSnippet> WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED! </CodeSnippet>
      <p>
        This is caused by the fact that every time you modify your SSH keys, the SSH server on your
        VMs regenerates its host keys. If you are using the VPN you are safe. Just follow the error
        message on how to update the host key on your local machine.
      </p>
    </HelpParagraph>
  </HelpPage>
</template>
