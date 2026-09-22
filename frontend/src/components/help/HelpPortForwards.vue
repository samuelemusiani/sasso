<script setup lang="ts">
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        By defaults, VMs are only accessible with the VPN, but sometimes you may want to access them
        directly from the internet. For example, if you have a web server running on a VM, you may
        want to make it accessible from the internet without the need of connecting to the VPN
        first.
      </p>
      <p>
        Port forwards are a way to make a VM accessible from the internet by forwarding a port from
        the host to a port on the VM.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Public IP and Out Port</template>
      <p>
        After the port forward, you will be able to access a specified port on the VM by connecting
        to the public IP of the host and the out port of your port forward.
      </p>
      <p>
        It's important to note that the <b>public IP is chosen by the system administrator</b>,
        while the <b>out port is chosen randomly</b>. You can only specify the IP and the port of
        the VM that you want to forward.
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Fields</template>
      <p>
        <b class="text-info">Destination Port</b> is the port on the VM that you want to forward. It
        can be any any port greater than 0.
      </p>
      <p>
        <b class="text-info">Destination IP</b> is the IP of the VM that you want to forward. It
        must be an IP that belongs to one of your networks and has to differ from both the network,
        gateway and broadcast address.
      </p>
      <p>
        <b class="text-info">Approved</b> is the status of the port forward. Port forwards are not
        automatically working, but they need to be approved by one of the system administrators. It
        will work only if approved is "true".
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Procol</template>
      <p>The port forward protocol will be both TCP and UDP. You cannot choose the protocol.</p>
    </HelpParagraph>
  </HelpPage>
</template>
