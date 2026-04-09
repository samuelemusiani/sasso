<script setup lang="ts">
import { getStatusClass } from '@/const'
import { RouterLink } from 'vue-router'
import HelpParagraph from '@/components/help/HelpParagraph.vue'
import HelpPage from '@/components/help/HelpPage.vue'
</script>

<template>
  <HelpPage>
    <HelpParagraph>
      <template #title>Introduction</template>
      <p>
        Networks are used to connect your VMs to the internet and to each other. In this page you
        can manage your networks and see a summary of all of them.
      </p>
      <p>
        You can think of a network as a switch with a dedicated router that allows the VMs to
        connect to the internet.
      </p>
      <p>
        VMs also need <b>interfaces</b> to connect to the intenet, but each interface be placed on
        an underlying network. This is like attaching a the network card of a computer directly to
        the switch (your network).
      </p>
      <p>
        VMs can talk to each other only on the same network and no one outside the network can talk
        to the VMs inside the network (unless a
        <RouterLink to="/port-forwards" class="link link-primary">port forward</RouterLink> is set
        up).
      </p>
    </HelpParagraph>

    <HelpParagraph>
      <template #title>Fields</template>
      <p>
        <b class="text-info">Name</b> is the name of your network. Use it to distinguish between
        your networks.
      </p>
      <p>
        <b class="text-info">Group</b>: if "Me" the network belongs to you, otherwise it belongs to
        the group specified.
      </p>
      <p>
        <b class="text-info">Status</b>: Networks can have multiple different states. If a state is
        prefixed with "<i>Pre-</i>" it means that the network is waiting to enter the state without
        the prefix. For example, "Pre-Creating" means that the network is waiting to be actually
        created and enter the "Creating" state.
      </p>
      <ul class="list-disc pl-6">
        <li>
          <b :class="getStatusClass('pre-creating')">Pre-Creating</b>: the network is waiting to be
          created.
        </li>
        <li>
          <b :class="getStatusClass('creating')">Creating</b>: the network is being created. This
          process can take a while, be patient!
        </li>
        <li>
          <b :class="getStatusClass('reconfiguring')">Reconfiguring</b>: the network is being
          reconfigured. This process happens if you toggle the "VLAN support" option.
        </li>
        <li>
          <b :class="getStatusClass('running')">Ready</b>: the network is ready to be used. You can
          attach your VMs to it by adding an interface to them.
        </li>
        <li>
          <b :class="getStatusClass('pre-deleting')">Pre-Deleting</b>: the network is waiting to be
          deleted.
        </li>
        <li><b :class="getStatusClass('deleting')">Deleting</b>: the network is being deleted.</li>
        <li>
          <b :class="getStatusClass('unknown')">Unknown</b>: the network is in an unknown state.
          Contact the administrator if you see this.
        </li>
      </ul>
      <p>
        <b class="text-info">VLAN Support</b>: As a network is like a switch, it can support VLANs.
        If VLAN support is enabled, when you add an interface to a VM you can specify a VLAN ID for
        the interface. This allows you to have multiple isolated networks on the same underlying
        network. If VLAN support is disabled, all interfaces on the network will be on the same VLAN
        and you won't be able to specify a VLAN ID for the interfaces.
        <b>The gateway is on VLAN 0</b>.
      </p>
      <p>
        <b class="text-info">Subnet</b> it's the subnet of the network. In the VMs interfaces you
        can put whatever IP address you want, but ONLY the IPs in the subnet will be able to talk to
        the gateway and to the internet.
      </p>
      <p><b class="text-info">Gateway</b> ip address of the gateway.</p>
    </HelpParagraph>
  </HelpPage>
</template>
